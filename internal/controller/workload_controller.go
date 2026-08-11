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
	"go.emeland.io/modelsrv/pkg/model/system"
)

// WorkloadReconciler maps native K8s workload resources to ComponentInstance.
// It handles Deployment, StatefulSet, DaemonSet, CronJob, and Job.
type WorkloadReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Model    model.Model
	Index    *NameIndex
	RuleEval *RuleEvaluation

	prototype client.Object
	kind      string
	skipFunc  func(client.Object) bool
}

// NewWorkloadReconciler creates a reconciler for the given workload type.
func NewWorkloadReconciler(c client.Client, scheme *runtime.Scheme, m model.Model, idx *NameIndex, prototype client.Object, kind string, skipFunc func(client.Object) bool) *WorkloadReconciler {
	return &WorkloadReconciler{
		Client:    c,
		Scheme:    scheme,
		Model:     m,
		Index:     idx,
		prototype: prototype,
		kind:      kind,
		skipFunc:  skipFunc,
	}
}

// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets;daemonsets,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs;cronjobs,verbs=get;list;watch

func (r *WorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContext(ctx)

	obj := r.prototype.DeepCopyObject().(client.Object)
	err := r.Get(ctx, req.NamespacedName, obj)

	if err == nil {
		if r.skipFunc != nil && r.skipFunc(obj) {
			return ctrl.Result{}, nil
		}
		ci, id := componentInstanceFromMeta(obj)
		if ci == nil {
			log.Error(nil, "skipping workload with no resolvable UUID", "kind", r.kind, "name", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		// If the Helm owner index knows about this resource, set the SystemInstance ref.
		if ci.GetSystemInstance() == nil {
			if siID := r.Index.GetHelmOwner(KindComponentInstance, req.NamespacedName.String()); siID != uuid.Nil {
				ci.SetSystemInstance(&system.SystemInstanceRef{InstanceId: siID})
			}
		}
		if err := r.Model.AddComponentInstance(ci); err != nil {
			log.Error(err, "could not add component instance to model", "kind", r.kind)
			return ctrl.Result{}, err
		}
		r.Index.Put(KindComponentInstance, req.NamespacedName.String(), id)
		r.reconcileComponentReferenceFindings(id, obj)
		r.RuleEval.run(obj)
	} else if k8serrors.IsNotFound(err) {
		id := r.Index.Delete(KindComponentInstance, req.NamespacedName.String())
		if id == uuid.Nil {
			return ctrl.Result{}, nil
		}
		// Clean up any reference findings for this resource.
		deleteReferenceFinding(r.Model, id, ReferencedResourceNotFound)
		deleteReferenceFinding(r.Model, id, MissingResourceReference)
		err = r.Model.DeleteComponentInstanceById(id)
		if errors.Is(err, common.ErrComponentInstanceNotFound) {
			err = nil
		}
	} else {
		log.Error(err, fmt.Sprintf("could not get %s %s", r.kind, req.NamespacedName))
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, err
}

func (r *WorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named(r.kind).
		For(r.prototype).
		Complete(r)
}

// reconcileComponentReferenceFindings checks the component-reference annotation
// and emits or clears findings accordingly.
func (r *WorkloadReconciler) reconcileComponentReferenceFindings(id uuid.UUID, obj client.Object) {
	annotations := obj.GetAnnotations()
	// Check both new and legacy annotation keys.
	compID := annotationUUID(annotations, AnnotationComponentReference)
	if compID == uuid.Nil {
		compID = annotationUUID(annotations, AnnotationComponentID)
	}

	if compID == uuid.Nil {
		// No reference annotation at all: emit MissingResourceReference.
		deleteReferenceFinding(r.Model, id, ReferencedResourceNotFound)
		upsertMissingResourceReference(r.Model, id, events.ComponentInstanceResource, obj.GetName(), AnnotationComponentReference)
		return
	}

	// Annotation present: clear MissingResourceReference.
	deleteReferenceFinding(r.Model, id, MissingResourceReference)

	// Check if the referenced Component exists in the local model.
	if r.Model.GetComponentById(compID) == nil {
		upsertReferencedResourceNotFound(r.Model, id, events.ComponentInstanceResource, compID, events.ComponentResource, obj.GetName())
	} else {
		deleteReferenceFinding(r.Model, id, ReferencedResourceNotFound)
	}
}
