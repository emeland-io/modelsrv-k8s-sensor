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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logr "sigs.k8s.io/controller-runtime/pkg/log"

	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/finding"
)

// MissingRoleSpecAnnotationFindingTypeID is a stable UUID for the finding type
// raised when a Role/ClusterRole lacks the emeland.io/k8s-sensor-role-spec-id annotation.
var MissingRoleSpecAnnotationFindingTypeID = uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")

// RoleReconciler maps K8s Role and ClusterRole resources to EmELand Role.
// Instantiated once per K8s kind (Role, ClusterRole).
type RoleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Model  model.Model
	Index  *NameIndex

	prototype          client.Object
	kind               string
	bindingReconcilers []*RoleBindingReconciler
}

// NewRoleReconciler creates a reconciler for a Role or ClusterRole kind.
func NewRoleReconciler(c client.Client, scheme *runtime.Scheme, m model.Model, idx *NameIndex, prototype client.Object, kind string) *RoleReconciler {
	return &RoleReconciler{
		Client:    c,
		Scheme:    scheme,
		Model:     m,
		Index:     idx,
		prototype: prototype,
		kind:      kind,
	}
}

// SetBindingReconciler adds a RoleBindingReconciler to be notified when a
// Role appears that has pending binding references.
func (r *RoleReconciler) SetBindingReconciler(rbc *RoleBindingReconciler) {
	r.bindingReconcilers = append(r.bindingReconcilers, rbc)
}

// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;clusterroles,verbs=get;list;watch

func (r *RoleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContext(ctx)

	obj := r.prototype.DeepCopyObject().(client.Object)
	err := r.Get(ctx, req.NamespacedName, obj)

	if err == nil {
		role, id := roleFromRBAC(obj)
		if role == nil {
			log.Error(nil, "skipping RBAC resource with no resolvable UUID", "kind", r.kind, "name", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		if err := r.Model.AddRole(role); err != nil {
			log.Error(err, "could not add role to model", "kind", r.kind)
			return ctrl.Result{}, err
		}
		indexKey := roleIndexKey(req.NamespacedName)
		r.Index.Put(KindRole, indexKey, id)

		// Issue a finding if the RoleSpec annotation is missing.
		r.reconcileRoleSpecFinding(id, obj)

		// Re-enqueue any bindings that arrived before this Role.
		for _, bindingName := range r.Index.ResolvePendingBindings(indexKey) {
			nn := parseNamespacedName(bindingName)
			for _, rbc := range r.bindingReconcilers {
				if rbc.IsClusterScoped() == (nn.Namespace == "") {
					rbc.EnqueueBinding(bindingName)
				}
			}
		}
	} else if k8serrors.IsNotFound(err) {
		indexKey := roleIndexKey(req.NamespacedName)
		id := r.Index.Delete(KindRole, indexKey)
		if id == uuid.Nil {
			return ctrl.Result{}, nil
		}
		// Remove associated finding.
		_ = r.Model.DeleteFindingById(roleSpecFindingID(id))

		err = r.Model.DeleteRole(id)
		if errors.Is(err, common.ErrRoleNotFound) {
			err = nil
		}
	} else {
		log.Error(err, fmt.Sprintf("could not get %s %s", r.kind, req.NamespacedName))
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, err
}

// reconcileRoleSpecFinding creates or removes a finding based on whether the
// RoleSpec annotation is present.
func (r *RoleReconciler) reconcileRoleSpecFinding(roleID uuid.UUID, obj client.Object) {
	specID := annotationUUID(obj.GetAnnotations(), AnnotationRoleSpecID)
	findingID := roleSpecFindingID(roleID)

	if specID != uuid.Nil {
		// Annotation is set -- remove any existing finding.
		_ = r.Model.DeleteFindingById(findingID)
		return
	}

	f := finding.NewFinding(findingID)
	f.SetDisplayName(fmt.Sprintf("Missing RoleSpec annotation on %s/%s", r.kind, obj.GetName()))
	f.SetDescription(fmt.Sprintf(
		"The %s %q does not have the %s annotation set to a valid RoleSpec UUID.",
		r.kind, obj.GetName(), AnnotationRoleSpecID,
	))
	f.SetFindingTypeById(MissingRoleSpecAnnotationFindingTypeID)
	f.SetResources([]*common.ResourceRef{{
		ResourceId:   roleID,
		ResourceType: events.RoleResource,
	}})
	_ = r.Model.AddFinding(f)
}

// roleSpecFindingID derives a deterministic finding UUID from a role UUID using
// UUID v5 in a private namespace.
func roleSpecFindingID(roleID uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(MissingRoleSpecAnnotationFindingTypeID, roleID[:])
}

func (r *RoleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named(r.kind).
		For(r.prototype).
		Complete(r)
}
