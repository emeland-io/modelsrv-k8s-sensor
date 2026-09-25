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

	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/common"

	"gitlab.com/emeland/k8s-model/internal/crdcheck"
)

// NamespaceReconciler reconciles Namespace objects into EmELand Context entities.
// The kube-system namespace becomes the root cluster context; all other namespaces
// become child contexts with kube-system as parent.
type NamespaceReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Model       model.Model
	Index       *NameIndex
	RuleEval    *RuleEvaluation
	CRDReporter *crdcheck.Reporter

	mu               sync.RWMutex
	clusterContextID uuid.UUID
}

// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch

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
	if err != nil {
		if k8serrors.IsNotFound(err) {
			if req.Name == "kube-system" {
				log.Info("kube-system namespace gone; clearing cluster Context for CRD findings")
				r.setClusterContextID(uuid.Nil)
				r.CRDReporter.ClearContext()
			}
			id := r.Index.Delete(KindContext, req.Name)
			if id != uuid.Nil {
				deleteReferenceFinding(r.Model, id, ReferencedResourceNotFound)
				err = r.Model.DeleteContextById(id)
				if errors.Is(err, common.ErrContextNotFound) {
					err = nil
				}
			}
			return ctrl.Result{}, err
		}
		log.Error(err, fmt.Sprintf("could not get Namespace %s", req.Name))
		return ctrl.Result{}, err
	}

	clusterID := r.getClusterContextID()

	if ns.Name != "kube-system" && clusterID == uuid.Nil {
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}

	emCtx, id := convertNamespaceToContext(ns, clusterID)
	if emCtx == nil {
		log.Error(nil, "skipping Namespace with no resolvable UUID", "name", req.Name)
		return ctrl.Result{}, nil
	}

	if ns.Name == "kube-system" {
		r.setClusterContextID(id)
	}

	if err := r.Model.AddContext(emCtx); err != nil {
		log.Error(err, "could not add context to model")
		return ctrl.Result{}, err
	}
	r.Index.Put(KindContext, req.Name, id)
	if ns.Name == "kube-system" {
		log.Info("kube-system modeled; attaching CRDNotAvailable findings to cluster Context",
			"clusterContextID", id.String(),
		)
		r.CRDReporter.ReportWithContext(id)
	}
	r.reconcileContextParentFinding(id, ns)
	r.RuleEval.run(ns)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Complete(r)
}

// crdMissingFallback attaches CRDNotAvailable findings to the sensor Node
// when kube-system is not in the cluster (so there is no cluster Context).
type crdMissingFallback struct {
	Client   client.Client
	Reporter *crdcheck.Reporter
}

// Start runs after the manager cache has synced. If kube-system exists,
// NamespaceReconciler will attach findings to the cluster Context; this
// only fires the Node fallback when that namespace is absent.
func (f *crdMissingFallback) Start(ctx context.Context) error {
	log := logr.FromContext(ctx).WithName("crdcheck-fallback")
	if f == nil || f.Reporter == nil {
		log.Info("CRD missing fallback runnable skipped; reporter is nil")
		return nil
	}
	ns := &corev1.Namespace{}
	if err := f.Client.Get(ctx, client.ObjectKey{Name: "kube-system"}, ns); err != nil {
		log.Info("kube-system not found after cache sync; attaching CRD findings to sensor Node",
			"err", err.Error(),
		)
		f.Reporter.ReportFallback()
		return nil
	}
	log.Info("kube-system present after cache sync; NamespaceReconciler will attach CRD findings to cluster Context",
		"uid", string(ns.UID),
	)
	return nil
}

// SetupCRDMissingFallback registers a runnable that falls back to the sensor
// Node when kube-system never appears. No-op when reporter is nil.
func SetupCRDMissingFallback(mgr ctrl.Manager, reporter *crdcheck.Reporter) error {
	if reporter == nil {
		return nil
	}
	return mgr.Add(&crdMissingFallback{
		Client:   mgr.GetClient(),
		Reporter: reporter,
	})
}

// reconcileContextParentFinding checks the context-parent annotation and emits
// or clears a ReferencedResourceNotFound finding. This annotation is optional,
// so no MissingResourceReference is emitted when it is absent.
func (r *NamespaceReconciler) reconcileContextParentFinding(id uuid.UUID, ns *corev1.Namespace) {
	parentID := annotationUUID(ns.Annotations, AnnotationContextParent)
	if parentID == uuid.Nil {
		// Annotation not set -- no finding needed (it's optional).
		deleteReferenceFinding(r.Model, id, ReferencedResourceNotFound)
		return
	}

	// Annotation present: check if the referenced Context exists.
	if r.Model.GetContextById(parentID) == nil {
		upsertReferencedResourceNotFound(r.Model, id, events.ContextResource, parentID, events.ContextResource, ns.Name)
	} else {
		deleteReferenceFinding(r.Model, id, ReferencedResourceNotFound)
	}
}
