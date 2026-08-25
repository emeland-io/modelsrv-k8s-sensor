// Package crdcheck probes the K8s discovery API for expected CRDs and
// reports which ones are missing without failing startup.
package crdcheck

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/finding"
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
func Check(ctx context.Context, client discovery.DiscoveryInterface, checklist []CRDEntry) CheckResult {
	var result CheckResult

	// Fetch all server resources once to avoid per-CRD round-trips.
	// ServerGroupsAndResources can return partial results alongside an error
	// (e.g. when some API groups are unreachable). We use whatever data we get.
	_, resourceLists, err := client.ServerGroupsAndResources()
	if err != nil && resourceLists == nil {
		// Total discovery failure with no usable data at all.
		result.DiscoveryErr = err
		result.Missing = append(result.Missing, checklist...)
		return result
	}
	if err != nil {
		// Partial failure: some groups unreachable, but we got data for others.
		result.DiscoveryErr = err
	}

	available := make(map[string]struct{})
	for _, rl := range resourceLists {
		if rl == nil {
			continue
		}
		for _, r := range rl.APIResources {
			// Key: "group/version/resource"
			key := fmt.Sprintf("%s/%s", rl.GroupVersion, r.Name)
			available[key] = struct{}{}
		}
	}

	for _, entry := range checklist {
		key := fmt.Sprintf("%s/%s/%s", entry.Group, entry.Version, entry.Resource)
		if _, ok := available[key]; ok {
			result.Available = append(result.Available, entry)
		} else {
			result.Missing = append(result.Missing, entry)
		}
	}
	return result
}

// LogAndReport logs missing CRDs and creates Findings in the model.
// It does not return an error; missing CRDs are informational, not fatal.
func LogAndReport(log logr.Logger, m model.Model, result CheckResult) {
	if result.DiscoveryErr != nil {
		log.Error(result.DiscoveryErr, "CRD discovery returned partial results, some groups may be unreachable")
	}

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

	// Create findings for missing CRDs.
	ensureCRDMissingFindingType(m)
	for _, entry := range result.Missing {
		createCRDMissingFinding(m, entry)
	}
}

const crdMissingFindingKind = "CRDNotAvailable"

func ensureCRDMissingFindingType(m model.Model) {
	kind := finding.FindingKind(crdMissingFindingKind)
	id := finding.TypeIDForKind(kind)
	if ft := m.GetFindingTypeById(id); ft != nil {
		return
	}
	ft := finding.NewFindingType(id)
	ft.SetDisplayName(crdMissingFindingKind)
	ft.SetDescription("A CRD expected by the k8s sensor is not installed in the cluster.")
	_ = m.AddFindingType(ft)
}

func createCRDMissingFinding(m model.Model, entry CRDEntry) {
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
	_ = m.AddFinding(f)
}
