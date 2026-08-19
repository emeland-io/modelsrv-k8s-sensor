/*
Copyright 2025 Lutz Behnke <lutz.behnke@emeland.io>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logr "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/finding"
)

// MissingSubjectAnnotationFindingTypeID is a stable UUID for the finding type
// raised when a RoleBinding/ClusterRoleBinding lacks the emeland.io/k8s-sensor-subject-id annotation.
var MissingSubjectAnnotationFindingTypeID = uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d480")

// RoleBindingReconciler maps K8s RoleBinding and ClusterRoleBinding to EmELand Binding.
type RoleBindingReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Model     model.Model
	Index     *NameIndex
	Whitelist *RBACWhitelist

	prototype client.Object
	kind      string

	// requeue is fed by EnqueueBinding to re-reconcile bindings whose Role
	// arrived late (after the binding was first reconciled).
	requeue chan event.GenericEvent
}

// NewRoleBindingReconciler creates a reconciler for a RoleBinding or ClusterRoleBinding kind.
func NewRoleBindingReconciler(c client.Client, scheme *runtime.Scheme, m model.Model, idx *NameIndex, prototype client.Object, kind string, wl *RBACWhitelist) *RoleBindingReconciler {
	return &RoleBindingReconciler{
		Client:    c,
		Scheme:    scheme,
		Model:     m,
		Index:     idx,
		Whitelist: wl,
		prototype: prototype,
		kind:      kind,
		requeue:   make(chan event.GenericEvent, 64),
	}
}

// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings;clusterrolebindings,verbs=get;list;watch

func (r *RoleBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContext(ctx)

	obj := r.prototype.DeepCopyObject().(client.Object)
	err := r.Get(ctx, req.NamespacedName, obj)

	if err == nil {
		roleRefKind, roleRefName, subjects := r.extractRoleBindingFields(obj)
		subjectKind := firstSubjectKind(subjects)

		// Build the index key for the referenced role. Namespaced Roles are
		// stored as "namespace/name"; cluster-scoped ClusterRoles as just "name".
		roleIndexKey := roleRefName
		if roleRefKind == "Role" && obj.GetNamespace() != "" {
			roleIndexKey = obj.GetNamespace() + "/" + roleRefName
		}

		binding, id := bindingFromRBAC(obj, roleIndexKey, subjectKind, r.Index)
		if binding == nil {
			log.Error(nil, "skipping RBAC binding with no resolvable UUID", "kind", r.kind, "name", req.NamespacedName)
			return ctrl.Result{}, nil
		}

		// If the referenced Role is not in the index yet, record this binding
		// as pending so the Role controller can trigger re-reconciliation later.
		if r.Index.Get(KindRole, roleIndexKey) == uuid.Nil {
			r.Index.AddPendingBinding(roleIndexKey, req.NamespacedName.String())
		} else {
			// Role is present now, clear any stale pending entry.
			r.Index.RemovePendingBinding(roleIndexKey, req.NamespacedName.String())
		}

		if err := r.Model.AddBinding(binding); err != nil {
			log.Error(err, "could not add binding to model", "kind", r.kind)
			return ctrl.Result{}, err
		}
		r.Index.Put(KindBinding, req.NamespacedName.String(), id)

		// Issue a finding if the subject annotation is missing.
		r.reconcileSubjectFinding(id, obj)
	} else if k8serrors.IsNotFound(err) {
		id := r.Index.Delete(KindBinding, req.NamespacedName.String())
		if id == uuid.Nil {
			return ctrl.Result{}, nil
		}
		// Clean up any pending-binding entry (in case role never arrived).
		r.Index.RemovePendingBindingByName(req.NamespacedName.String())
		// Remove associated finding.
		_ = r.Model.DeleteFindingById(subjectFindingID(id))

		err = r.Model.DeleteBinding(id)
		if errors.Is(err, common.ErrBindingNotFound) {
			err = nil
		}
	} else {
		log.Error(err, fmt.Sprintf("could not get %s %s", r.kind, req.NamespacedName))
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, err
}

// extractRoleBindingFields returns the roleRef kind, roleRef name, and subjects
// list from the underlying RoleBinding or ClusterRoleBinding.
func (r *RoleBindingReconciler) extractRoleBindingFields(obj client.Object) (string, string, []rbacv1.Subject) {
	switch rb := obj.(type) {
	case *rbacv1.RoleBinding:
		return rb.RoleRef.Kind, rb.RoleRef.Name, rb.Subjects
	case *rbacv1.ClusterRoleBinding:
		return rb.RoleRef.Kind, rb.RoleRef.Name, rb.Subjects
	}
	return "", "", nil
}

// reconcileSubjectFinding creates or removes a finding based on whether the
// subject annotation is present.
func (r *RoleBindingReconciler) reconcileSubjectFinding(bindingID uuid.UUID, obj client.Object) {
	// Skip finding creation for whitelisted resources.
	if r.Whitelist.IsWhitelisted(r.kind, obj.GetName()) {
		_ = r.Model.DeleteFindingById(subjectFindingID(bindingID))
		return
	}

	subjectID := annotationUUID(obj.GetAnnotations(), AnnotationSubjectID)
	findingID := subjectFindingID(bindingID)

	if subjectID != uuid.Nil {
		// Annotation is set -- remove any existing finding.
		_ = r.Model.DeleteFindingById(findingID)
		return
	}

	f := finding.NewFinding(findingID)
	f.SetDisplayName(fmt.Sprintf("Missing subject annotation on %s/%s", r.kind, obj.GetName()))
	f.SetDescription(fmt.Sprintf(
		"The %s %q does not have the %s annotation set to a valid Group or Identity UUID.",
		r.kind, obj.GetName(), AnnotationSubjectID,
	))
	f.SetFindingTypeById(MissingSubjectAnnotationFindingTypeID)
	f.SetResources([]*common.ResourceRef{{
		ResourceId:   bindingID,
		ResourceType: events.BindingResource,
	}})
	_ = r.Model.AddFinding(f)
}

// subjectFindingID derives a deterministic finding UUID from a binding UUID.
func subjectFindingID(bindingID uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(MissingSubjectAnnotationFindingTypeID, bindingID[:])
}

func (r *RoleBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named(r.kind).
		For(r.prototype).
		WatchesRawSource(source.Channel(r.requeue, &handler.EnqueueRequestForObject{})).
		Complete(r)
}

// EnqueueBinding triggers a re-reconcile of the given binding (identified by
// its NamespacedName string, e.g. "namespace/name" or "name").
func (r *RoleBindingReconciler) EnqueueBinding(nameKey string) {
	nn := parseNamespacedName(nameKey)
	// Best-effort enqueue; drop if channel is full (will be picked up on
	// the next periodic resync).
	select {
	case r.requeue <- event.GenericEvent{Object: &metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{Name: nn.Name, Namespace: nn.Namespace},
	}}:
	default:
	}
}

// IsClusterScoped returns true if this reconciler handles cluster-scoped
// resources (ClusterRoleBinding).
func (r *RoleBindingReconciler) IsClusterScoped() bool {
	return r.kind == "ClusterRoleBinding"
}

// parseNamespacedName splits a "namespace/name" string back into a
// types.NamespacedName. This is the inverse of NamespacedName.String(), which
// always formats as "namespace/name" with exactly one slash. Cluster-scoped
// resources have an empty namespace, so their string form is just "name" (no
// slash) after going through roleIndexKey/our index storage.
func parseNamespacedName(s string) types.NamespacedName {
	for i := range s {
		if s[i] == '/' {
			return types.NamespacedName{Namespace: s[:i], Name: s[i+1:]}
		}
	}
	return types.NamespacedName{Name: s}
}
