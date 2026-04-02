package controller

import (
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
	"gitlab.com/emeland/k8s-model/internal/model"
)

// Annotation keys used to link native K8s resources to EmELand entities.
const (
	AnnotationComponentID      = "componentId.emeland.io"
	AnnotationSystemInstanceID = "systemInstanceId.emeland.io"
	AnnotationAPIID            = "apiId.emeland.io"
	AnnotationSystemID         = "systemId.emeland.io"
)

// --- CRD helpers ---

func parseVersion(v v1alpha1.Version) model.Version {
	return model.Version{
		Version:        v.Version,
		AvailableFrom:  parseDate(v.AvailableFrom),
		DeprecatedFrom: parseDate(v.DeprecatedFrom),
		TerminatedFrom: parseDate(v.TerminatedFrom),
	}
}

func parseDate(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil
	}
	return &t
}

// parseSystemRef parses a SystemRef from either a systemId or a VersionRef.
func parseSystemRef(sysId string, sysRef *v1alpha1.VersionRef) *model.SystemRef {
	if sysId != "" {
		uid, err := uuid.Parse(sysId)
		if err == nil {
			return &model.SystemRef{SystemId: uid}
		}
	}
	if sysRef != nil && sysRef.Name != "" && sysRef.Version != "" {
		return &model.SystemRef{
			SystemRef: &model.EntityVersion{
				Name:    sysRef.Name,
				Version: sysRef.Version,
			},
		}
	}
	return nil
}

// --- Shared helpers for native K8s resources ---

// parseOptionalUUID parses a UUID string, returning uuid.Nil if empty or invalid.
func parseOptionalUUID(s string) uuid.UUID {
	if s == "" {
		return uuid.Nil
	}
	uid, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return uid
}

// uuidFromMeta converts metadata.uid to a uuid.UUID.
func uuidFromMeta(obj metav1.ObjectMeta) uuid.UUID {
	return parseOptionalUUID(string(obj.UID))
}

// copyAnnotations copies all annotations into a new map.
func copyAnnotations(src map[string]string) map[string]string {
	ann := make(map[string]string, len(src))
	for k, v := range src {
		ann[k] = v
	}
	return ann
}

// annotationUUID parses a UUID from an annotation value, returning uuid.Nil if absent or invalid.
func annotationUUID(annotations map[string]string, key string) uuid.UUID {
	return parseOptionalUUID(annotations[key])
}

// IsOwnedByCronJob returns true if the object has an ownerReference of kind CronJob.
func IsOwnedByCronJob(obj client.Object) bool {
	for _, ref := range obj.GetOwnerReferences() {
		if ref.Kind == "CronJob" {
			return true
		}
	}
	return false
}

// componentInstanceFromMeta builds a ComponentInstance from any K8s object's metadata.
func componentInstanceFromMeta(obj client.Object) *model.ComponentInstance {
	uid := parseOptionalUUID(string(obj.GetUID()))
	if uid == uuid.Nil {
		return nil
	}

	annotations := obj.GetAnnotations()
	ci := &model.ComponentInstance{
		DisplayName: obj.GetName(),
		InstanceId:  uid,
		Annotations: copyAnnotations(annotations),
	}

	if siID := annotationUUID(annotations, AnnotationSystemInstanceID); siID != uuid.Nil {
		ci.SystemInstance = &model.SystemInstanceRef{InstanceId: siID}
	}

	return ci
}

// apiInstanceFromMeta builds an APIInstance from any K8s object's metadata.
func apiInstanceFromMeta(obj client.Object) *model.APIInstance {
	uid := parseOptionalUUID(string(obj.GetUID()))
	if uid == uuid.Nil {
		return nil
	}

	annotations := obj.GetAnnotations()
	ai := &model.APIInstance{
		DisplayName: obj.GetName(),
		InstanceId:  uid,
		Annotations: copyAnnotations(annotations),
	}

	if apiID := annotationUUID(annotations, AnnotationAPIID); apiID != uuid.Nil {
		ai.ApiRef = model.ApiRef{ApiID: apiID}
	}

	if siID := annotationUUID(annotations, AnnotationSystemInstanceID); siID != uuid.Nil {
		ai.SystemInstance = &model.SystemInstanceRef{InstanceId: siID}
	}

	return ai
}
