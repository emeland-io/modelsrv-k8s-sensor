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
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logr "sigs.k8s.io/controller-runtime/pkg/log"

	"gitlab.com/emeland/k8s-model/internal/model"
)

// IngressReconciler reconciles an Ingress object
type IngressReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Model  model.Model
}

//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=get

func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContext(ctx)

	ing := &networkingv1.Ingress{}
	err := r.Get(ctx, req.NamespacedName, ing)

	if err == nil {
		ai := convertIngressToAPIInstance(ing)
		if ai == nil {
			return ctrl.Result{}, nil
		}
		err = r.Model.AddApiInstance(ai, req.NamespacedName.String(), nil)
		if err != nil {
			log.Error(err, "could not add api instance to model")
		}
	} else if errors.IsNotFound(err) {
		err = r.Model.DeleteApiInstanceByResourceName(req.NamespacedName.String())
		if err == model.ApiInstanceNotFoundError {
			err = nil
		}
	} else {
		log.Error(err, fmt.Sprintf("could not get Ingress %s", req.NamespacedName))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.Ingress{}).
		Complete(r)
}

func convertIngressToAPIInstance(ing *networkingv1.Ingress) *model.APIInstance {
	uid := uuidFromMeta(ing.ObjectMeta)
	if uid == uuid.Nil {
		return nil
	}

	ai := &model.APIInstance{
		DisplayName: ing.Name,
		InstanceId:  uid,
		Annotations: copyAnnotations(ing.ObjectMeta),
	}

	if apiID := annotationUUID(ing.ObjectMeta, AnnotationAPIID); apiID != uuid.Nil {
		ai.ApiRef = model.ApiRef{ApiID: apiID}
	}

	if siID := annotationUUID(ing.ObjectMeta, AnnotationSystemInstanceID); siID != uuid.Nil {
		ai.SystemInstance = &model.SystemInstanceRef{InstanceId: siID}
	}

	return ai
}
