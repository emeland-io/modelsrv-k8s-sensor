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

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logr "sigs.k8s.io/controller-runtime/pkg/log"

	"gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
	"gitlab.com/emeland/k8s-model/internal/model"
)

// SystemReconciler reconciles a System object
type SystemReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Model  model.Model
}

// +kubebuilder:rbac:groups=structure.emeland.io,resources=systems,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=structure.emeland.io,resources=systems/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=structure.emeland.io,resources=systems/finalizers,verbs=update

func (r *SystemReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContext(ctx)

	sys := &v1alpha1.System{}
	err := r.Get(ctx, req.NamespacedName, sys)

	if err == nil {
		err = r.Model.AddSystem(convertSystem(sys), req.NamespacedName.String(), r.Client.Status())
		if err != nil {
			log.Error(err, "could not add service to model")
		}
	} else if errors.IsNotFound(err) {
		err = r.Model.DeleteSystemByResourceName(req.NamespacedName.String())
		if err == model.SystemNotFoundError {
			err = nil // ignore a resource that is not even in the model
		}
	} else {
		log.Error(err, fmt.Sprintf("could not get System %s", req.NamespacedName))
	}

	return ctrl.Result{}, err

}

// SetupWithManager sets up the controller with the Manager.
func (r *SystemReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.System{}).
		Complete(r)
}

func convertSystem(sys *v1alpha1.System) *model.System {
	newSys := &model.System{
		DisplayName: sys.Spec.DisplayName,
		Description: sys.Spec.Description,
	}

	// parse Version
	newSys.Version = model.ParseVersion(sys.Spec.Version)

	// parse ID if set
	if sys.Spec.SystemId != "" {
		uid, err := uuid.Parse(sys.Spec.SystemId)
		if err == nil {
			newSys.SystemId = &uid
		}
	}

	// transfer annotations
	newSys.Annotations = make(map[string]string)
	for key, value := range sys.Annotations {
		newSys.Annotations[key] = value
	}

	return newSys
}
