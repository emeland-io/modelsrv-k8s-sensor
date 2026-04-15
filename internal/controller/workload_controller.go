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

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logr "sigs.k8s.io/controller-runtime/pkg/log"

	"gitlab.com/emeland/k8s-model/internal/model"
)

// WorkloadReconciler maps native K8s workload resources to ComponentInstance.
// It handles Deployment, StatefulSet, DaemonSet, CronJob, and Job.
//
//+kubebuilder:rbac:groups=apps,resources=deployments;statefulsets;daemonsets,verbs=get;list;watch
//+kubebuilder:rbac:groups=batch,resources=jobs;cronjobs,verbs=get;list;watch
type WorkloadReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Model  model.Model

	// prototype is a zero-value object used for Get() and For() (e.g. &appsv1.Deployment{}).
	prototype client.Object
	// kind is the human-readable resource kind for log messages.
	kind string
	// skipFunc optionally filters out resources that should not be tracked.
	skipFunc func(client.Object) bool
}

// NewWorkloadReconciler creates a reconciler for the given workload type.
func NewWorkloadReconciler(c client.Client, scheme *runtime.Scheme, m model.Model, prototype client.Object, kind string, skipFunc func(client.Object) bool) *WorkloadReconciler {
	return &WorkloadReconciler{
		Client:    c,
		Scheme:    scheme,
		Model:     m,
		prototype: prototype,
		kind:      kind,
		skipFunc:  skipFunc,
	}
}

func (r *WorkloadReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContext(ctx)

	obj := r.prototype.DeepCopyObject().(client.Object)
	err := r.Get(ctx, req.NamespacedName, obj)

	if err == nil {
		if r.skipFunc != nil && r.skipFunc(obj) {
			return ctrl.Result{}, nil
		}
		ci := componentInstanceFromMeta(obj)
		if ci == nil {
			return ctrl.Result{}, nil
		}
		if err := r.Model.AddComponentInstance(ci, req.NamespacedName.String(), nil); err != nil {
			log.Error(err, "could not add component instance to model", "kind", r.kind)
			return ctrl.Result{}, err
		}
	} else if k8serrors.IsNotFound(err) {
		err = r.Model.DeleteComponentInstanceByResourceName(req.NamespacedName.String())
		if errors.Is(err, model.ErrComponentInstanceNotFound) {
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
