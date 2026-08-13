package controller

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/finding"
)

// Annotation keys for linking native K8s resources to EmELand entities.
// These are parsed by the respective reconcilers to set first-class references
// and emit findings when the referenced resource is missing.
const (
	// AnnotationAPIReference is the UUID of the EmELand API resource that an
	// ApiInstance (K8s Service) references.
	AnnotationAPIReference = "emeland.io/k8s-sensor/api-reference"

	// AnnotationComponentReference is the UUID of the EmELand Component resource
	// that a ComponentInstance (K8s Deployment, StatefulSet, DaemonSet, CronJob, Job)
	// references.
	AnnotationComponentReference = "emeland.io/k8s-sensor/component-reference"

	// AnnotationContextParent is the UUID of an EmELand Context resource that a
	// Context (K8s Namespace) references as its parent. This is only for
	// non-cluster parent relationships (e.g. application grouping). The
	// namespace-to-cluster relationship is implied and does not need this annotation.
	AnnotationContextParent = "emeland.io/k8s-sensor/context-parent"
)

// Finding kinds matching the definitions in modelsrv PR #153. The string values
// must match exactly so that TypeIDForKind produces the same stable UUIDs on
// both sides.
const (
	ReferencedResourceNotFound finding.FindingKind = "ReferencedResourceNotFound"
	MissingResourceReference   finding.FindingKind = "MissingResourceReference"
)

// findingNamespace is the UUID v5 namespace for deriving deterministic finding
// IDs from (subject UUID + finding kind). This avoids duplicate findings when
// the same resource is reconciled multiple times.
var findingNamespace = uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef0123456789")

// referenceFindingID derives a stable finding UUID from the subject resource ID
// and the finding kind, so repeated reconciliation upserts rather than duplicates.
func referenceFindingID(subjectID uuid.UUID, kind finding.FindingKind) uuid.UUID {
	key := append([]byte(kind), subjectID[:]...)
	return uuid.NewSHA1(findingNamespace, key)
}

// upsertReferencedResourceNotFound emits a finding indicating that the subject
// references a UUID that does not exist in the local model. Resources layout:
// [subject, missing target].
func upsertReferencedResourceNotFound(m model.Model, subjectID uuid.UUID, subjectType events.ResourceType, targetID uuid.UUID, targetType events.ResourceType, subjectName string) {
	fID := referenceFindingID(subjectID, ReferencedResourceNotFound)
	f := finding.NewFinding(fID)
	f.SetDisplayName(fmt.Sprintf("Referenced resource not found for %s", subjectName))
	f.SetDescription(fmt.Sprintf(
		"The resource %s references %s which is not registered in the local model.",
		subjectName, targetID,
	))
	f.SetFindingTypeById(finding.TypeIDForKind(ReferencedResourceNotFound))
	f.SetResources([]*common.ResourceRef{
		{ResourceId: subjectID, ResourceType: subjectType},
		{ResourceId: targetID, ResourceType: targetType},
	})
	_ = m.AddFinding(f)
}

// upsertMissingResourceReference emits a finding indicating that the subject
// lacks a required EmELand reference annotation. Resources layout: [subject].
func upsertMissingResourceReference(m model.Model, subjectID uuid.UUID, subjectType events.ResourceType, subjectName, annotationKey string) {
	fID := referenceFindingID(subjectID, MissingResourceReference)
	f := finding.NewFinding(fID)
	f.SetDisplayName(fmt.Sprintf("Missing resource reference on %s", subjectName))
	f.SetDescription(fmt.Sprintf(
		"The resource %s does not have the %s annotation set.",
		subjectName, annotationKey,
	))
	f.SetFindingTypeById(finding.TypeIDForKind(MissingResourceReference))
	f.SetResources([]*common.ResourceRef{
		{ResourceId: subjectID, ResourceType: subjectType},
	})
	_ = m.AddFinding(f)
}

// deleteReferenceFinding removes a previously emitted finding for the given
// subject and kind, if it exists.
func deleteReferenceFinding(m model.Model, subjectID uuid.UUID, kind finding.FindingKind) {
	fID := referenceFindingID(subjectID, kind)
	_ = m.DeleteFindingById(fID)
}

// RegisterReferenceFindingTypes registers the well-known FindingType resources
// for reference-related findings in the model.
func RegisterReferenceFindingTypes(m model.Model) error {
	types := []struct {
		kind        finding.FindingKind
		displayName string
		description string
	}{
		{
			kind:        ReferencedResourceNotFound,
			displayName: "ReferencedResourceNotFound",
			description: "A resource references another resource by UUID that is not registered in the local model.",
		},
		{
			kind:        MissingResourceReference,
			displayName: "MissingResourceReference",
			description: "A resource lacks a required EmELand reference to another resource.",
		},
	}

	for _, t := range types {
		id := finding.TypeIDForKind(t.kind)
		ft := finding.NewFindingType(id)
		ft.SetDisplayName(t.displayName)
		ft.SetDescription(t.description)
		if err := m.AddFindingType(ft); err != nil {
			return err
		}
	}
	return nil
}
