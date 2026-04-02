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

// DaemonSetReconciler reconciles a DaemonSet object
type DaemonSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Model  model.Model
}

//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=daemonsets/status,verbs=get

func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContext(ctx)

	ds := &appsv1.DaemonSet{}
	err := r.Get(ctx, req.NamespacedName, ds)

	if err == nil {
		ci := convertDaemonSetToComponentInstance(ds)
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
		log.Error(err, fmt.Sprintf("could not get DaemonSet %s", req.NamespacedName))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DaemonSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.DaemonSet{}).
		Complete(r)
}

func convertDaemonSetToComponentInstance(ds *appsv1.DaemonSet) *model.ComponentInstance {
	uid := uuidFromMeta(ds.ObjectMeta)
	if uid == uuid.Nil {
		return nil
	}

	ci := &model.ComponentInstance{
		DisplayName: ds.Name,
		InstanceId:  uid,
		Annotations: copyAnnotations(ds.ObjectMeta),
	}

	if siID := annotationUUID(ds.ObjectMeta, AnnotationSystemInstanceID); siID != uuid.Nil {
		ci.SystemInstance = &model.SystemInstanceRef{InstanceId: siID}
	}

	return ci
}
