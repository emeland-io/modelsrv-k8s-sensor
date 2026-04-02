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
	"sync"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logr "sigs.k8s.io/controller-runtime/pkg/log"

	"gitlab.com/emeland/k8s-model/internal/model"
)

// NamespaceReconciler reconciles Namespace objects into EmELand Context entities.
// The kube-system namespace becomes the root cluster context; all other namespaces
// become child contexts with kube-system as parent.
type NamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Model  model.Model

	mu               sync.RWMutex
	clusterContextID uuid.UUID
}

//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch

func (r *NamespaceReconciler) getClusterContextID() uuid.UUID {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clusterContextID
}

func (r *NamespaceReconciler) setClusterContextID(id uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clusterContextID = id
}

func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logr.FromContext(ctx)

	ns := &corev1.Namespace{}
	err := r.Get(ctx, req.NamespacedName, ns)

	if err == nil {
		clusterID := r.getClusterContextID()

		// If the root context is not yet known and this is not kube-system, re-queue.
		if ns.Name != "kube-system" && clusterID == uuid.Nil {
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}

		emCtx := convertNamespaceToContext(ns, clusterID)
		if emCtx == nil {
			return ctrl.Result{}, nil
		}

		if ns.Name == "kube-system" {
			r.setClusterContextID(emCtx.ContextId)
		}

		if err := r.Model.AddContext(emCtx, req.Name); err != nil {
			log.Error(err, "could not add context to model")
			return ctrl.Result{}, err
		}
	} else if k8serrors.IsNotFound(err) {
		err = r.Model.DeleteContextByResourceName(req.Name)
		if errors.Is(err, model.ErrContextNotFound) {
			err = nil
		}
	} else {
		log.Error(err, fmt.Sprintf("could not get Namespace %s", req.Name))
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Complete(r)
}

func convertNamespaceToContext(ns *corev1.Namespace, clusterContextID uuid.UUID) *model.Context {
	uid := uuidFromMeta(ns.ObjectMeta)
	if uid == uuid.Nil {
		return nil
	}

	emCtx := &model.Context{
		DisplayName: ns.Name,
		ContextId:   uid,
		Description: fmt.Sprintf("Kubernetes namespace %s", ns.Name),
		Annotations: copyAnnotations(ns.ObjectMeta),
	}

	// kube-system is the root cluster context; all others are children.
	if ns.Name != "kube-system" && clusterContextID != uuid.Nil {
		emCtx.ParentId = clusterContextID
	}

	return emCtx
}
