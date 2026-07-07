package controller

import (
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/component"
	mdlctx "go.emeland.io/modelsrv/pkg/model/context"
	"go.emeland.io/modelsrv/pkg/model/system"
)

// Annotation keys used to link native K8s resources to EmELand entities.
const (
	AnnotationComponentID      = "componentId.emeland.io"
	AnnotationSystemInstanceID = "systemInstanceId.emeland.io"
	AnnotationAPIID            = "apiId.emeland.io"
	AnnotationSystemID         = "systemId.emeland.io"
)

// --- CRD helpers ---

func parseVersion(v v1alpha1.Version) common.Version {
	return common.Version{
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

func parseSystemRef(sysId string, sysRef *v1alpha1.VersionRef) *system.SystemRef {
	if sysId != "" {
		uid, err := uuid.Parse(sysId)
		if err == nil && uid != uuid.Nil {
			return &system.SystemRef{SystemId: uid}
		}
	}
	// modelsrv SystemRef is UUID-only; VersionRef name/version cannot be represented.
	_ = sysRef
	return nil
}

func parseAPIRefs(refs []v1alpha1.APIRef) []mdlapi.ApiRef {
	if len(refs) == 0 {
		return nil
	}
	out := make([]mdlapi.ApiRef, 0, len(refs))
	for _, ref := range refs {
		if id := parseOptionalUUID(ref.ApiId); id != uuid.Nil {
			out = append(out, mdlapi.ApiRef{ApiID: id})
			continue
		}
		if ref.VersionRef.Name != "" && ref.VersionRef.Version != "" {
			out = append(out, mdlapi.ApiRef{
				ApiRef: &common.EntityVersion{
					Name:    ref.VersionRef.Name,
					Version: ref.VersionRef.Version,
				},
			})
		}
	}
	return out
}

// --- Shared helpers for native K8s resources ---

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

func resourceID(specID string, meta metav1.ObjectMeta) uuid.UUID {
	if id := parseOptionalUUID(specID); id != uuid.Nil {
		return id
	}
	return uuidFromMeta(meta)
}

func uuidFromMeta(obj metav1.ObjectMeta) uuid.UUID {
	return parseOptionalUUID(string(obj.UID))
}

func applyAnnotations(ann interface{ Add(string, string) }, src map[string]string) {
	for k, v := range src {
		ann.Add(k, v)
	}
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

func convertSystem(sys *v1alpha1.System) (system.System, uuid.UUID) {
	id := resourceID(sys.Spec.SystemId, sys.ObjectMeta)
	if id == uuid.Nil {
		return nil, uuid.Nil
	}
	s := system.NewSystem(id)
	s.SetDisplayName(sys.Spec.DisplayName)
	s.SetDescription(sys.Spec.Description)
	s.SetVersion(parseVersion(sys.Spec.Version))
	s.SetAbstract(sys.Spec.Abstract)
	applyAnnotations(s.GetAnnotations(), sys.Annotations)
	return s, id
}

func convertAPI(api *v1alpha1.API) (mdlapi.API, uuid.UUID) {
	id := resourceID(api.Spec.ApiId, api.ObjectMeta)
	if id == uuid.Nil {
		return nil, uuid.Nil
	}
	a := mdlapi.NewAPI(id)
	a.SetDisplayName(api.Spec.DisplayName)
	a.SetDescription(api.Spec.Description)
	a.SetVersion(parseVersion(api.Spec.Version))
	if apiType, err := mdlapi.ParseApiType(api.Spec.Type); err == nil {
		a.SetType(apiType)
	}
	if sysRef := parseSystemRef(api.Spec.SystemId, &api.Spec.SystemRef); sysRef != nil {
		a.SetSystem(sysRef)
	}
	applyAnnotations(a.GetAnnotations(), api.Annotations)
	return a, id
}

func convertComponent(comp *v1alpha1.Component) (component.Component, uuid.UUID) {
	id := resourceID(comp.Spec.ComponentId, comp.ObjectMeta)
	if id == uuid.Nil {
		return nil, uuid.Nil
	}
	c := component.NewComponent(id)
	c.SetDisplayName(comp.Spec.DisplayName)
	c.SetDescription(comp.Spec.Description)
	c.SetVersion(parseVersion(comp.Spec.Version))
	if sysRef := parseSystemRef(comp.Spec.SystemId, &comp.Spec.SystemRef); sysRef != nil {
		c.SetSystem(sysRef)
	}
	if consumes := parseAPIRefs(comp.Spec.Consumes); len(consumes) > 0 {
		c.SetConsumes(consumes)
	}
	if provides := parseAPIRefs(comp.Spec.Provides); len(provides) > 0 {
		c.SetProvides(provides)
	}
	applyAnnotations(c.GetAnnotations(), comp.Annotations)
	return c, id
}

func convertSystemInstance(sysInst *v1alpha1.SystemInstance) (system.SystemInstance, uuid.UUID) {
	id := resourceID(sysInst.Spec.InstanceId, sysInst.ObjectMeta)
	if id == uuid.Nil {
		return nil, uuid.Nil
	}
	si := system.NewSystemInstance(id)
	si.SetDisplayName(sysInst.Spec.DisplayName)
	if sysRef := parseSystemRef(sysInst.Spec.SystemId, nil); sysRef != nil {
		si.SetSystemRef(sysRef)
	}
	applyAnnotations(si.GetAnnotations(), sysInst.Annotations)
	return si, id
}

func convertNamespaceToContext(ns *corev1.Namespace, clusterContextID uuid.UUID) (mdlctx.Context, uuid.UUID) {
	id := uuidFromMeta(ns.ObjectMeta)
	if id == uuid.Nil {
		return nil, uuid.Nil
	}
	ctx := mdlctx.NewContext(id)
	ctx.SetDisplayName(ns.Name)
	ctx.SetDescription("Kubernetes namespace " + ns.Name)
	if ns.Name != "kube-system" && clusterContextID != uuid.Nil {
		ctx.SetParentById(clusterContextID)
	}
	applyAnnotations(ctx.GetAnnotations(), ns.Annotations)
	return ctx, id
}

func componentInstanceFromMeta(obj client.Object) (component.ComponentInstance, uuid.UUID) {
	uid := parseOptionalUUID(string(obj.GetUID()))
	if uid == uuid.Nil {
		return nil, uuid.Nil
	}
	ci := component.NewComponentInstance(uid)
	ci.SetDisplayName(obj.GetName())
	applyAnnotations(ci.GetAnnotations(), obj.GetAnnotations())
	if siID := annotationUUID(obj.GetAnnotations(), AnnotationSystemInstanceID); siID != uuid.Nil {
		ci.SetSystemInstance(&system.SystemInstanceRef{InstanceId: siID})
	}
	return ci, uid
}

func apiInstanceFromMeta(obj client.Object) (mdlapi.ApiInstance, uuid.UUID) {
	uid := parseOptionalUUID(string(obj.GetUID()))
	if uid == uuid.Nil {
		return nil, uuid.Nil
	}
	ai := mdlapi.NewApiInstance(uid)
	ai.SetDisplayName(obj.GetName())
	applyAnnotations(ai.GetAnnotations(), obj.GetAnnotations())
	annotations := obj.GetAnnotations()
	if apiID := annotationUUID(annotations, AnnotationAPIID); apiID != uuid.Nil {
		ai.SetApiRef(&mdlapi.ApiRef{ApiID: apiID})
	}
	if siID := annotationUUID(annotations, AnnotationSystemInstanceID); siID != uuid.Nil {
		ai.SetSystemInstance(&system.SystemInstanceRef{InstanceId: siID})
	}
	return ai, uid
}

func annotationUUID(annotations map[string]string, key string) uuid.UUID {
	return parseOptionalUUID(annotations[key])
}
