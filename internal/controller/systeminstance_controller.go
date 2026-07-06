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

	"gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/common"
)

// SystemInstanceReconciler reconciles a SystemInstance object
type SystemInstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Model  model.Model
	Index  *NameIndex
}

// +kubebuilder:rbac:groups=structure.emeland.io,resources=systeminstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=structure.emeland.io,resources=systeminstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=structure.emeland.io,resources=systeminstances/finalizers,verbs=update

func (r *SystemInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContext(ctx)

	systemInstance := &v1alpha1.SystemInstance{}
	err := r.Get(ctx, req.NamespacedName, systemInstance)

	if err == nil {
		obj, id := convertSystemInstance(systemInstance)
		if obj == nil {
			log.Error(nil, "skipping SystemInstance with no resolvable UUID", "name", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		if err = r.Model.AddSystemInstance(obj); err != nil {
			log.Error(err, "could not add systemInstance to model")
			return ctrl.Result{}, err
		}
		r.Index.Put(KindSystemInstance, req.NamespacedName.String(), id)
	} else if k8serrors.IsNotFound(err) {
		id := r.Index.Delete(KindSystemInstance, req.NamespacedName.String())
		if id == uuid.Nil {
			log.Error(nil, "SystemInstance deleted but no UUID in index", "name", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		err = r.Model.DeleteSystemInstanceById(id)
		if errors.Is(err, common.ErrSystemInstanceNotFound) {
			err = nil
		}
	} else {
		log.Error(err, fmt.Sprintf("could not get SystemInstance %s", req.NamespacedName))
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *SystemInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.SystemInstance{}).
		Complete(r)
}
