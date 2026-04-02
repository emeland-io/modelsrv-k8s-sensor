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

// APIReconciler reconciles a API object
type APIReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Model  model.Model
}

// +kubebuilder:rbac:groups=structure.emeland.io,resources=apis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=structure.emeland.io,resources=apis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=structure.emeland.io,resources=apis/finalizers,verbs=update

func (r *APIReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContext(ctx)

	api := &v1alpha1.API{}
	err := r.Get(ctx, req.NamespacedName, api)

	if err == nil {
		err = r.Model.AddApi(convertAPI(api), req.NamespacedName.String(), r.Client.Status())
		if err != nil {
			log.Error(err, "could not add api to model")
		}
	} else if errors.IsNotFound(err) {
		err = r.Model.DeleteApiByResourceName(req.NamespacedName.String())
		if err == model.ApiNotFoundError {
			err = nil // ignore a resource that is not even in the model
		}
	} else {
		log.Error(err, fmt.Sprintf("could not get API %s", req.NamespacedName))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *APIReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.API{}).
		Complete(r)
}

func convertAPI(api *v1alpha1.API) *model.API {
	newApi := &model.API{
		DisplayName: api.Spec.DisplayName,
		Description: api.Spec.Description,
		Version:     parseVersion(api.Spec.Version),
		Type:        model.ParseApiType(api.Spec.Type),
		System:      parseSystemRef(api.Spec.SystemId, &api.Spec.SystemRef),
	}

	// parse ID if set
	if api.Spec.ApiId != "" {
		uid, err := uuid.Parse(api.Spec.ApiId)
		if err == nil {
			newApi.ApiId = uid
		}
	}

	// transfer annotations
	newApi.Annotations = make(map[string]string)
	for key, value := range api.Annotations {
		newApi.Annotations[key] = value
	}

	return newApi
}
