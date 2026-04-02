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
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logr "sigs.k8s.io/controller-runtime/pkg/log"

	"gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
	"gitlab.com/emeland/k8s-model/internal/model"
)

// ComponentReconciler reconciles a Component object
type ComponentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Model  model.Model
}

// +kubebuilder:rbac:groups=structure.emeland.io,resources=components,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=structure.emeland.io,resources=components/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=structure.emeland.io,resources=components/finalizers,verbs=update

func (r *ComponentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContext(ctx)

	comp := &v1alpha1.Component{}
	err := r.Get(ctx, req.NamespacedName, comp)

	if err == nil {
		err = r.Model.AddComponent(convertComponent(comp), req.NamespacedName.String(), r.Client.Status())
		if err != nil {
			log.Error(err, "could not add component to model")
		}
	} else if errors.IsNotFound(err) {
		err = r.Model.DeleteComponentByResourceName(req.NamespacedName.String())
		if err == model.ComponentNotFoundError {
			err = nil // ignore a resource that is not even in the model
		}
	} else {
		log.Error(err, fmt.Sprintf("could not get Component %s", req.NamespacedName))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ComponentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Component{}).
		Complete(r)
}

func convertComponent(comp *v1alpha1.Component) *model.Component {
	return &model.Component{
		DisplayName: comp.Spec.DisplayName,
		Description: comp.Spec.Description,
		ComponentId: parseOptionalUUID(comp.Spec.ComponentId),
		Version:     parseVersion(comp.Spec.Version),
		System:      parseSystemRef(comp.Spec.SystemId, &comp.Spec.SystemRef),
		Annotations: copyAnnotations(comp.ObjectMeta),
	}
}
