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
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logr "sigs.k8s.io/controller-runtime/pkg/log"

	"gitlab.com/emeland/k8s-model/internal/model"
)

// JobReconciler reconciles a Job object
type JobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Model  model.Model
}

//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch
//+kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get

func (r *JobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContext(ctx)

	job := &batchv1.Job{}
	err := r.Get(ctx, req.NamespacedName, job)

	if err == nil {
		// Skip Jobs owned by a CronJob — the CronJob controller handles those.
		if isOwnedByCronJob(job.ObjectMeta) {
			return ctrl.Result{}, nil
		}

		ci := convertJobToComponentInstance(job)
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
		log.Error(err, fmt.Sprintf("could not get Job %s", req.NamespacedName))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.Job{}).
		Complete(r)
}

func convertJobToComponentInstance(job *batchv1.Job) *model.ComponentInstance {
	uid := uuidFromMeta(job.ObjectMeta)
	if uid == uuid.Nil {
		return nil
	}

	ci := &model.ComponentInstance{
		DisplayName: job.Name,
		InstanceId:  uid,
		Annotations: copyAnnotations(job.ObjectMeta),
	}

	if siID := annotationUUID(job.ObjectMeta, AnnotationSystemInstanceID); siID != uuid.Nil {
		ci.SystemInstance = &model.SystemInstanceRef{InstanceId: siID}
	}

	return ci
}
