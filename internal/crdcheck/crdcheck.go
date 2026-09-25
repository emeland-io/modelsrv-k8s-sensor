// Package crdcheck probes the K8s discovery API for expected CRDs and
// reports which ones are missing without failing startup.
package crdcheck

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/finding"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
)

// CRDEntry describes one expected CRD to check for.
type CRDEntry struct {
	// Group is the API group (e.g. "cert-manager.io").
	Group string
	// Version is the preferred API version (e.g. "v1").
	Version string
	// Resource is the plural resource name (e.g. "certificates").
	Resource string
	// DisplayName is a human-readable label for logging and findings.
	DisplayName string
	// Category groups the CRD for reporting (e.g. "CertManager", "Grafana Operator").
	Category string
}

// String returns "group/version/resource".
func (e CRDEntry) String() string {
	return fmt.Sprintf("%s/%s/%s", e.Group, e.Version, e.Resource)
}

// DefaultChecklist is the built-in set of CRDs the sensor expects to find.
// Operators can override this via --crd-checklist.
var DefaultChecklist = []CRDEntry{
	// EmELand native
	{Group: "structure.emeland.io", Version: "v1alpha1", Resource: "systems", DisplayName: "EmELand System", Category: "EmELand"},
	{Group: "structure.emeland.io", Version: "v1alpha1", Resource: "apis", DisplayName: "EmELand API", Category: "EmELand"},
	{Group: "structure.emeland.io", Version: "v1alpha1", Resource: "components", DisplayName: "EmELand Component", Category: "EmELand"},
	{Group: "structure.emeland.io", Version: "v1alpha1", Resource: "systeminstances", DisplayName: "EmELand SystemInstance", Category: "EmELand"},
	{Group: "structure.emeland.io", Version: "v1alpha1", Resource: "findingrules", DisplayName: "EmELand FindingRule", Category: "EmELand"},

	// Helm: not included because the sensor detects Helm releases via core/v1
	// Secrets of type helm.sh/release.v1, not via a Helm-specific CRD.

	// Cert-Manager
	{Group: "cert-manager.io", Version: "v1", Resource: "certificates", DisplayName: "Certificate", Category: "CertManager"},
	{Group: "cert-manager.io", Version: "v1", Resource: "issuers", DisplayName: "Issuer", Category: "CertManager"},
	{Group: "cert-manager.io", Version: "v1", Resource: "clusterissuers", DisplayName: "ClusterIssuer", Category: "CertManager"},

	// Prometheus Operator
	{Group: "monitoring.coreos.com", Version: "v1", Resource: "servicemonitors", DisplayName: "ServiceMonitor", Category: "Prometheus Operator"},
	{Group: "monitoring.coreos.com", Version: "v1", Resource: "prometheusrules", DisplayName: "PrometheusRule", Category: "Prometheus Operator"},

	// Grafana Operator
	{Group: "grafana.integreatly.org", Version: "v1beta1", Resource: "grafanadashboards", DisplayName: "GrafanaDashboard", Category: "Grafana Operator"},
}

// ParseChecklist parses a comma-separated list of CRD entries in the format
// "group/version/resource". Each entry is looked up in DefaultChecklist for
// metadata; unknown entries get a generic DisplayName and Category.
func ParseChecklist(raw string) ([]CRDEntry, error) {
	if raw == "" {
		return nil, nil
	}

	// Build lookup from DefaultChecklist.
	lookup := make(map[string]CRDEntry, len(DefaultChecklist))
	for _, e := range DefaultChecklist {
		lookup[e.String()] = e
	}

	var result []CRDEntry
	for _, token := range strings.Split(raw, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if known, ok := lookup[token]; ok {
			result = append(result, known)
		} else {
			entry, err := parseCRDToken(token)
			if err != nil {
				return nil, fmt.Errorf("invalid CRD entry %q: %w", token, err)
			}
			result = append(result, entry)
		}
	}
	return result, nil
}

func parseCRDToken(token string) (CRDEntry, error) {
	// Expected format: group/version/resource
	parts := strings.SplitN(token, "/", 4)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return CRDEntry{}, fmt.Errorf("expected format group/version/resource")
	}

	return CRDEntry{
		Group:       parts[0],
		Version:     parts[1],
		Resource:    parts[2],
		DisplayName: parts[2],
		Category:    parts[0],
	}, nil
}

// CheckResult holds the outcome of a CRD availability check.
type CheckResult struct {
	Available []CRDEntry
	Missing   []CRDEntry
	// DiscoveryErr is set when discovery returned partial results alongside
	// an error (e.g. some API groups were unreachable). The check still
	// proceeds with whatever data was returned, but callers should log this.
	DiscoveryErr error
}

// Check probes the cluster's discovery API for each entry in checklist.
// It does not fail on missing CRDs; those are returned in result.Missing.
func Check(ctx context.Context, log logr.Logger, client discovery.DiscoveryInterface, checklist []CRDEntry) CheckResult {
	var result CheckResult
	if log.GetSink() == nil {
		log = logr.Discard()
	}
	log = log.WithName("crdcheck")

	log.Info("probing discovery API for expected CRDs",
		"checklistSize", len(checklist),
		"checklist", crdEntryStrings(checklist),
	)

	// Fetch all server resources once to avoid per-CRD round-trips.
	// ServerGroupsAndResources can return partial results alongside an error
	// (e.g. when some API groups are unreachable). We use whatever data we get.
	groups, resourceLists, err := client.ServerGroupsAndResources()
	log.Info("ServerGroupsAndResources returned",
		"err", errString(err),
		"errType", fmt.Sprintf("%T", err),
		"apiGroupCount", len(groups),
		"resourceListCount", len(resourceLists),
		"resourceListsNil", resourceLists == nil,
	)
	logDiscoveryGroups(log, groups)
	logDiscoveryError(log, err, resourceLists)

	if err != nil && resourceLists == nil {
		// Total discovery failure with no usable data at all.
		result.DiscoveryErr = err
		result.Missing = append(result.Missing, checklist...)
		log.Error(err, "total discovery failure; treating entire checklist as missing",
			"missing", crdEntryStrings(result.Missing),
		)
		return result
	}
	if err != nil {
		// Partial failure: some groups unreachable, but we got data for others.
		result.DiscoveryErr = err
		log.Info("partial discovery failure; continuing with returned resource lists")
	}

	available := make(map[string]struct{})
	var discoveredKeys []string
	for _, rl := range resourceLists {
		if rl == nil {
			log.Info("skipping nil APIResourceList from discovery")
			continue
		}
		names := make([]string, 0, len(rl.APIResources))
		for _, r := range rl.APIResources {
			// Key: "group/version/resource"
			key := fmt.Sprintf("%s/%s", rl.GroupVersion, r.Name)
			available[key] = struct{}{}
			discoveredKeys = append(discoveredKeys, key)
			names = append(names, r.Name)
		}
		log.Info("discovery APIResourceList",
			"groupVersion", rl.GroupVersion,
			"resourceCount", len(rl.APIResources),
			"resources", names,
		)
		log.V(1).Info("discovery APIResourceList details",
			"groupVersion", rl.GroupVersion,
			"resources", formatAPIResources(rl.APIResources),
		)
	}
	sort.Strings(discoveredKeys)
	log.V(1).Info("flattened discovery keys",
		"count", len(discoveredKeys),
		"keys", discoveredKeys,
	)

	for _, entry := range checklist {
		key := fmt.Sprintf("%s/%s/%s", entry.Group, entry.Version, entry.Resource)
		if _, ok := available[key]; ok {
			result.Available = append(result.Available, entry)
			log.Info("expected CRD is available",
				"crd", key,
				"displayName", entry.DisplayName,
				"category", entry.Category,
			)
			continue
		}
		result.Missing = append(result.Missing, entry)
		sameGroup := keysWithPrefix(discoveredKeys, entry.Group+"/")
		sameGV := keysWithPrefix(discoveredKeys, entry.Group+"/"+entry.Version+"/")
		log.Info("expected CRD not in discovery result",
			"crd", key,
			"displayName", entry.DisplayName,
			"category", entry.Category,
			"groupPresent", len(sameGroup) > 0,
			"groupVersionPresent", len(sameGV) > 0,
			"sameGroupKeys", sameGroup,
			"sameGroupVersionKeys", sameGV,
		)
	}
	return result
}

// Log writes the CRD availability outcome. Missing CRDs are informational, not fatal.
func Log(log logr.Logger, result CheckResult) {
	if log.GetSink() == nil {
		log = logr.Discard()
	}
	log = log.WithName("crdcheck")

	if result.DiscoveryErr != nil {
		log.Error(result.DiscoveryErr, "CRD discovery returned an error (partial results may still have been used)",
			"errType", fmt.Sprintf("%T", result.DiscoveryErr),
		)
		logDiscoveryError(log, result.DiscoveryErr, nil)
	}

	log.Info("CRD availability summary",
		"availableCount", len(result.Available),
		"missingCount", len(result.Missing),
		"available", crdEntryStrings(result.Available),
		"missing", crdEntryStrings(result.Missing),
		"discoveryErr", errString(result.DiscoveryErr),
	)

	if len(result.Missing) == 0 {
		log.Info("all expected CRDs available", "count", len(result.Available))
		return
	}

	for _, entry := range result.Missing {
		log.Info("expected CRD not available in cluster",
			"crd", entry.String(),
			"displayName", entry.DisplayName,
			"category", entry.Category,
		)
	}
	log.Info("CRD availability check complete",
		"available", len(result.Available),
		"missing", len(result.Missing),
	)
}

// Report creates CRDNotAvailable findings attached to subject. subject must
// identify an EmELand resource that already exists in the local model
// (typically the cluster Context). Findings are not created when subject is
// empty — they would otherwise land in landscape with no resources.
func Report(log logr.Logger, m model.Model, result CheckResult, subject common.ResourceRef) {
	if log.GetSink() == nil {
		log = logr.Discard()
	}
	log = log.WithName("crdcheck")

	if len(result.Missing) == 0 {
		log.Info("skipping CRDNotAvailable findings; checklist is fully available",
			"availableCount", len(result.Available),
		)
		return
	}
	if subject.ResourceId == uuid.Nil {
		log.Info("skipping CRDNotAvailable findings; subject ResourceId is empty",
			"missingCount", len(result.Missing),
			"subjectType", subject.ResourceType,
		)
		return
	}

	log.Info("emitting CRDNotAvailable findings",
		"missingCount", len(result.Missing),
		"missing", crdEntryStrings(result.Missing),
		"subjectId", subject.ResourceId.String(),
		"subjectType", subject.ResourceType,
	)
	ensureCRDMissingFindingType(log, m)
	for _, entry := range result.Missing {
		createCRDMissingFinding(log, m, entry, subject)
	}
}

// Reporter upserts CRDNotAvailable findings, preferring the cluster Context
// and falling back to the sensor Node if kube-system never appears.
type Reporter struct {
	log      logr.Logger
	model    model.Model
	result   CheckResult
	fallback *common.ResourceRef

	mu                  sync.Mutex
	reportedWithContext bool
}

// NewReporter holds a check result for later finding emission. fallbackNodeID
// is the sensor Node used when the cluster Context is unavailable.
func NewReporter(log logr.Logger, m model.Model, result CheckResult, fallbackNodeID uuid.UUID) *Reporter {
	var fallback *common.ResourceRef
	if fallbackNodeID != uuid.Nil {
		fallback = &common.ResourceRef{
			ResourceId:   fallbackNodeID,
			ResourceType: events.NodeResource,
		}
	}
	if log.GetSink() == nil {
		log = logr.Discard()
	}
	log = log.WithName("crdcheck")
	r := &Reporter{
		log:      log,
		model:    m,
		result:   result,
		fallback: fallback,
	}
	log.Info("CRD finding reporter created",
		"missingCount", len(result.Missing),
		"missing", crdEntryStrings(result.Missing),
		"availableCount", len(result.Available),
		"fallbackNodeID", fallbackID(fallback),
		"fallbackType", fallbackType(fallback),
	)
	return r
}

// ReportWithContext attaches missing-CRD findings to the cluster Context.
// Safe to call on a nil Reporter.
func (r *Reporter) ReportWithContext(clusterContextID uuid.UUID) {
	if r == nil {
		return
	}
	if clusterContextID == uuid.Nil {
		r.log.Info("ReportWithContext skipped; cluster context ID is empty")
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.log.Info("attaching CRDNotAvailable findings to cluster Context",
		"clusterContextID", clusterContextID.String(),
		"alreadyReported", r.reportedWithContext,
		"missingCount", len(r.result.Missing),
	)
	Report(r.log, r.model, r.result, common.ResourceRef{
		ResourceId:   clusterContextID,
		ResourceType: events.ContextResource,
	})
	r.reportedWithContext = true
}

// ReportFallback attaches missing-CRD findings to the sensor Node when the
// cluster Context has not been reported. No-op after ReportWithContext.
// Safe to call on a nil Reporter.
func (r *Reporter) ReportFallback() {
	if r == nil {
		return
	}
	if r.fallback == nil {
		r.log.Info("ReportFallback skipped; no sensor Node configured")
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.reportedWithContext {
		r.log.Info("ReportFallback skipped; findings already attached to cluster Context",
			"fallbackNodeID", r.fallback.ResourceId.String(),
		)
		return
	}
	r.log.Info("attaching CRDNotAvailable findings to sensor Node (kube-system absent)",
		"fallbackNodeID", r.fallback.ResourceId.String(),
		"missingCount", len(r.result.Missing),
	)
	Report(r.log, r.model, r.result, *r.fallback)
}

// ClearContext drops the cluster-Context attachment (e.g. kube-system deleted)
// and re-emits findings on the sensor Node if one was configured.
// Safe to call on a nil Reporter.
func (r *Reporter) ClearContext() {
	if r == nil {
		return
	}
	r.log.Info("cluster Context cleared; re-attaching CRD findings to fallback if needed")
	r.mu.Lock()
	r.reportedWithContext = false
	r.mu.Unlock()
	r.ReportFallback()
}

const crdMissingFindingKind = "CRDNotAvailable"

func ensureCRDMissingFindingType(log logr.Logger, m model.Model) {
	kind := finding.FindingKind(crdMissingFindingKind)
	id := finding.TypeIDForKind(kind)
	if ft := m.GetFindingTypeById(id); ft != nil {
		log.V(1).Info("CRDNotAvailable finding type already registered", "findingTypeId", id.String())
		return
	}
	ft := finding.NewFindingType(id)
	ft.SetDisplayName(crdMissingFindingKind)
	ft.SetDescription("A CRD expected by the k8s sensor is not installed in the cluster.")
	if err := m.AddFindingType(ft); err != nil {
		log.Error(err, "unable to add CRDNotAvailable finding type", "findingTypeId", id.String())
		return
	}
	log.Info("registered CRDNotAvailable finding type", "findingTypeId", id.String())
}

func createCRDMissingFinding(log logr.Logger, m model.Model, entry CRDEntry, subject common.ResourceRef) {
	kind := finding.FindingKind(crdMissingFindingKind)
	typeID := finding.TypeIDForKind(kind)

	// Use a deterministic UUID from the CRD identity so re-runs upsert instead of duplicate.
	id := uuid.NewSHA1(typeID, []byte(entry.String()))

	f := finding.NewFinding(id)
	f.SetFindingTypeById(typeID)
	f.SetDisplayName(fmt.Sprintf("CRD not available: %s", entry.DisplayName))
	f.SetDescription(fmt.Sprintf(
		"The CRD %s (category: %s) is not installed in this cluster. "+
			"The sensor cannot watch resources of this type.",
		entry.String(), entry.Category,
	))
	f.SetResources([]*common.ResourceRef{{
		ResourceId:   subject.ResourceId,
		ResourceType: subject.ResourceType,
	}})
	if err := m.AddFinding(f); err != nil {
		log.Error(err, "unable to add CRD missing finding",
			"crd", entry.String(),
			"findingId", id.String(),
			"subjectId", subject.ResourceId.String(),
			"subjectType", subject.ResourceType,
		)
		return
	}
	log.Info("upserted CRDNotAvailable finding",
		"crd", entry.String(),
		"displayName", f.GetDisplayName(),
		"findingId", id.String(),
		"findingTypeId", typeID.String(),
		"subjectId", subject.ResourceId.String(),
		"subjectType", subject.ResourceType,
	)
}

func logDiscoveryError(log logr.Logger, err error, resourceLists []*metav1.APIResourceList) {
	if err == nil {
		return
	}
	log.Error(err, "discovery error details",
		"errType", fmt.Sprintf("%T", err),
		"resourceListCount", len(resourceLists),
		"resourceListsNil", resourceLists == nil,
	)
	var groupErr *discovery.ErrGroupDiscoveryFailed
	if errors.As(err, &groupErr) {
		log.Info("ErrGroupDiscoveryFailed: some API groups were unreachable (sidecar/proxy/mesh is a common cause)",
			"failedGroupCount", len(groupErr.Groups),
		)
		for gv, gerr := range groupErr.Groups {
			log.Error(gerr, "API group discovery failed",
				"group", gv.Group,
				"version", gv.Version,
				"groupVersion", gv.String(),
			)
		}
		return
	}
	log.V(1).Info("discovery error is not ErrGroupDiscoveryFailed",
		"errType", fmt.Sprintf("%T", err),
		"err", err.Error(),
	)
}

func logDiscoveryGroups(log logr.Logger, groups []*metav1.APIGroup) {
	if len(groups) == 0 {
		log.Info("discovery returned no API groups")
		return
	}
	names := make([]string, 0, len(groups))
	for _, g := range groups {
		if g == nil {
			continue
		}
		versions := make([]string, 0, len(g.Versions))
		for _, v := range g.Versions {
			versions = append(versions, v.GroupVersion)
		}
		log.V(1).Info("discovery API group",
			"name", g.Name,
			"preferredVersion", g.PreferredVersion.GroupVersion,
			"versions", versions,
		)
		names = append(names, g.Name)
	}
	log.Info("discovery API groups", "count", len(names), "groups", names)
}

func formatAPIResources(resources []metav1.APIResource) []string {
	out := make([]string, 0, len(resources))
	for _, r := range resources {
		out = append(out, fmt.Sprintf("name=%s kind=%s namespaced=%t verbs=%v",
			r.Name, r.Kind, r.Namespaced, r.Verbs))
	}
	return out
}

func crdEntryStrings(entries []CRDEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.String())
	}
	return out
}

func keysWithPrefix(keys []string, prefix string) []string {
	var out []string
	for _, k := range keys {
		if strings.HasPrefix(k, prefix) {
			out = append(out, k)
		}
	}
	return out
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func fallbackID(ref *common.ResourceRef) string {
	if ref == nil {
		return ""
	}
	return ref.ResourceId.String()
}

func fallbackType(ref *common.ResourceRef) events.ResourceType {
	if ref == nil {
		return events.UnknownResourceType
	}
	return ref.ResourceType
}
