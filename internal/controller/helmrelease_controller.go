package controller

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"gitlab.com/emeland/k8s-model/internal/helm"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/system"
)

// helmNamespaceUUID is a fixed namespace UUID for deterministic ID generation.
var helmNamespaceUUID = uuid.MustParse("a3e4c5d6-1234-5678-9abc-def012345678")

// HelmReleaseReconciler watches Secrets of type helm.sh/release.v1 and
// creates SystemInstance resources for each Helm release.
type HelmReleaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Model  model.Model
	Index  *NameIndex

	// latestRevision tracks the highest seen revision per release (namespace/name key).
	mu             sync.Mutex
	latestRevision map[string]int
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

func (r *HelmReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	secret := &corev1.Secret{}
	err := r.Get(ctx, req.NamespacedName, secret)

	if err == nil {
		// Filter: only process Helm release secrets.
		if secret.Type != "helm.sh/release.v1" {
			return ctrl.Result{}, nil
		}

		releaseName, revision, ok := helm.SecretNameParts(secret.Name)
		if !ok {
			log.V(1).Info("skipping secret with unparseable Helm name", "name", req.NamespacedName)
			return ctrl.Result{}, nil
		}

		// Only process the latest revision per release.
		releaseKey := req.Namespace + "/" + releaseName
		r.mu.Lock()
		if r.latestRevision == nil {
			r.latestRevision = make(map[string]int)
		}
		if existing, tracked := r.latestRevision[releaseKey]; tracked && revision < existing {
			r.mu.Unlock()
			return ctrl.Result{}, nil
		}
		r.latestRevision[releaseKey] = revision
		r.mu.Unlock()

		// Decode release data.
		releaseData, decErr := helm.DecodeReleaseData(secret.Data["release"])
		if decErr != nil {
			log.Error(decErr, "failed to decode Helm release data", "secret", req.NamespacedName)
			return ctrl.Result{}, nil
		}

		// Check if the release already deploys a SystemInstance CRD.
		resources := helm.ParseManifestResources(releaseData.Manifest)
		if helm.HasSystemInstance(resources) {
			log.V(1).Info("release already contains SystemInstance, skipping", "release", releaseName)
			return ctrl.Result{}, nil
		}

		// Build a deterministic UUID from namespace + release name.
		id := uuid.NewSHA1(helmNamespaceUUID, []byte(req.Namespace+"/"+releaseName))

		si := system.NewSystemInstance(id)
		si.SetDisplayName(releaseName)

		chartAnnotation := fmt.Sprintf("%s-%s", releaseData.Chart.Metadata.Name, releaseData.Chart.Metadata.Version)
		si.GetAnnotations().Add("helm.sh/chart", chartAnnotation)

		if err := r.Model.AddSystemInstance(si); err != nil {
			log.Error(err, "could not add SystemInstance to model", "release", releaseName)
			return ctrl.Result{}, err
		}
		r.Index.Put(KindSystemInstance, releaseKey, id)

		// Correlate resources deployed by this release to the SystemInstance.
		r.correlateResources(resources, req.Namespace, id, log)

		log.Info("created SystemInstance for Helm release", "release", releaseName, "revision", revision, "id", id)
		return ctrl.Result{}, nil
	}

	if k8serrors.IsNotFound(err) {
		// Secret was deleted. Parse name to figure out the release.
		releaseName, revision, ok := helm.SecretNameParts(req.Name)
		if !ok {
			return ctrl.Result{}, nil
		}

		releaseKey := req.Namespace + "/" + releaseName

		r.mu.Lock()
		tracked, exists := r.latestRevision[releaseKey]
		if !exists || revision != tracked {
			// This wasn't the latest revision we tracked, nothing to do.
			r.mu.Unlock()
			return ctrl.Result{}, nil
		}
		delete(r.latestRevision, releaseKey)
		r.mu.Unlock()

		// Delete the SystemInstance.
		id := r.Index.Delete(KindSystemInstance, releaseKey)
		if id == uuid.Nil {
			return ctrl.Result{}, nil
		}
		delErr := r.Model.DeleteSystemInstanceById(id)
		if delErr != nil && !errors.Is(delErr, common.ErrSystemInstanceNotFound) {
			log.Error(delErr, "could not delete SystemInstance from model", "release", releaseName)
			return ctrl.Result{}, delErr
		}
		log.Info("deleted SystemInstance for Helm release", "release", releaseName, "id", id)
		return ctrl.Result{}, nil
	}

	log.Error(err, "could not get Secret", "name", req.NamespacedName)
	return ctrl.Result{}, err
}

func (r *HelmReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("HelmRelease").
		For(&corev1.Secret{}, builder.WithPredicates(helmSecretPredicate())).
		Complete(r)
}

// helmSecretPredicate filters events to only Secrets of type helm.sh/release.v1.
func helmSecretPredicate() predicate.Funcs {
	isHelmSecret := func(obj client.Object) bool {
		secret, ok := obj.(*corev1.Secret)
		if !ok {
			return false
		}
		return secret.Type == "helm.sh/release.v1"
	}
	return predicate.NewPredicateFuncs(isHelmSecret)
}

// helmKindToResourceKind maps Helm manifest resource kinds to NameIndex ResourceKinds.
var helmKindToResourceKind = map[string]ResourceKind{
	"Deployment":  KindComponentInstance,
	"StatefulSet": KindComponentInstance,
	"DaemonSet":   KindComponentInstance,
	"Job":         KindComponentInstance,
	"CronJob":     KindComponentInstance,
	"Service":     KindAPIInstance,
	"Ingress":     KindAPIInstance,
}

// correlateResources links existing ComponentInstances and APIInstances
// to the SystemInstance created for this Helm release.
func (r *HelmReleaseReconciler) correlateResources(
	resources []helm.ManifestResource,
	releaseNamespace string,
	systemInstanceID uuid.UUID,
	log logr.Logger,
) {
	siRef := &system.SystemInstanceRef{InstanceId: systemInstanceID}

	for _, res := range resources {
		kind, ok := helmKindToResourceKind[res.Kind]
		if !ok {
			continue
		}

		ns := res.Namespace
		if ns == "" {
			ns = releaseNamespace
		}
		nameKey := ns + "/" + res.Name

		resourceID := r.Index.Get(kind, nameKey)
		if resourceID == uuid.Nil {
			// Resource not yet reconciled by the sensor; will be correlated
			// on next reconcile of that resource or next helm reconcile.
			continue
		}

		switch kind {
		case KindComponentInstance:
			ci := r.Model.GetComponentInstanceById(resourceID)
			if ci != nil {
				ci.SetSystemInstance(siRef)
				if err := r.Model.AddComponentInstance(ci); err != nil {
					log.Error(err, "could not update ComponentInstance with SystemInstance ref",
						"resource", nameKey, "systemInstance", systemInstanceID)
				}
			}
		case KindAPIInstance:
			ai := r.Model.GetApiInstanceById(resourceID)
			if ai != nil {
				ai.SetSystemInstance(siRef)
				if err := r.Model.AddApiInstance(ai); err != nil {
					log.Error(err, "could not update APIInstance with SystemInstance ref",
						"resource", nameKey, "systemInstance", systemInstanceID)
				}
			}
		}
	}
}
