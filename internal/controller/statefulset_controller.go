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

// StatefulSetReconciler reconciles a StatefulSet object
type StatefulSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Model  model.Model
}

//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=statefulsets/status,verbs=get

func (r *StatefulSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContext(ctx)

	ss := &appsv1.StatefulSet{}
	err := r.Get(ctx, req.NamespacedName, ss)

	if err == nil {
		ci := convertStatefulSetToComponentInstance(ss)
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
		log.Error(err, fmt.Sprintf("could not get StatefulSet %s", req.NamespacedName))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StatefulSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.StatefulSet{}).
		Complete(r)
}

func convertStatefulSetToComponentInstance(ss *appsv1.StatefulSet) *model.ComponentInstance {
	uid := uuidFromMeta(ss.ObjectMeta)
	if uid == uuid.Nil {
		return nil
	}

	ci := &model.ComponentInstance{
		DisplayName: ss.Name,
		InstanceId:  uid,
		Annotations: copyAnnotations(ss.ObjectMeta),
	}

	if siID := annotationUUID(ss.ObjectMeta, AnnotationSystemInstanceID); siID != uuid.Nil {
		ci.SystemInstance = &model.SystemInstanceRef{InstanceId: siID}
	}

	return ci
}
