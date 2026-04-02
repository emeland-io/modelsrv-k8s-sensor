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
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logr "sigs.k8s.io/controller-runtime/pkg/log"

	"gitlab.com/emeland/k8s-model/internal/model"
)

// DeploymentReconciler reconciles a Deployment object
type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Model  model.Model
}

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get

func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContext(ctx)

	dep := &appsv1.Deployment{}
	err := r.Get(ctx, req.NamespacedName, dep)

	if err == nil {
		ci := convertDeploymentToComponentInstance(dep)
		if ci == nil {
			return ctrl.Result{}, nil
		}
		err = r.Model.AddComponentInstance(ci, req.NamespacedName.String(), nil)
		if err != nil {
			log.Error(err, "could not add component instance to model")
		}
	} else if errors.IsNotFound(err) {
		err = r.Model.DeleteComponentInstanceByResourceName(req.NamespacedName.String())
		if err == model.ComponentInstanceNotFoundError {
			err = nil
		}
	} else {
		log.Error(err, fmt.Sprintf("could not get Deployment %s", req.NamespacedName))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Complete(r)
}

func convertDeploymentToComponentInstance(dep *appsv1.Deployment) *model.ComponentInstance {
	uid := uuidFromMeta(dep.ObjectMeta)
	if uid == uuid.Nil {
		return nil
	}

	ci := &model.ComponentInstance{
		DisplayName: dep.Name,
		InstanceId:  uid,
		Annotations: copyAnnotations(dep.ObjectMeta),
	}

	if compID := annotationUUID(dep.ObjectMeta, AnnotationComponentID); compID != uuid.Nil {
		ci.ComponentRef = model.EntityVersion{Name: dep.Name}
	}

	if siID := annotationUUID(dep.ObjectMeta, AnnotationSystemInstanceID); siID != uuid.Nil {
		ci.SystemInstance = &model.SystemInstanceRef{InstanceId: siID}
	}

	return ci
}
