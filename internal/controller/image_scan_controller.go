package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/artifact"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/finding"
)

// ImageNotRetrieved is raised when a container image is declared in the cluster
// but has not been retrieved successfully (ErrImagePull / ImagePullBackOff /
// InvalidImageName) and no container of that image has started.
const ImageNotRetrieved finding.FindingKind = "ImageNotRetrieved"

// clusterImageScanRequest is the fixed reconcile key for the cluster-wide scan.
var clusterImageScanRequest = types.NamespacedName{Name: "cluster"}

// ImageScanReconciler performs a cluster-wide scan of Pod container images and
// exposes them as Artifacts / ArtifactInstances in the landscape model.
type ImageScanReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Model  model.Model
	Index  *NameIndex
}

// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch

func (r *ImageScanReconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx).WithName("ImageScan")

	var podList corev1.PodList
	if err := r.List(ctx, &podList); err != nil {
		log.Error(err, "failed to list pods")
		return ctrl.Result{}, err
	}

	observations := collectImageObservations(podList.Items)
	desiredKeys := make(map[string]struct{}, len(observations))

	for key, obs := range observations {
		desiredKeys[key] = struct{}{}
		artID, err := r.ensureArtifact(obs)
		if err != nil {
			log.Error(err, "failed to ensure Artifact", "image", obs.displayName)
			return ctrl.Result{}, err
		}

		if obs.started {
			if err := r.ensureArtifactInstance(obs, artID); err != nil {
				log.Error(err, "failed to ensure ArtifactInstance", "image", obs.displayName)
				return ctrl.Result{}, err
			}
			deleteImageNotRetrievedFinding(r.Model, artID)
		} else {
			// No started copy: remove any previous instance for this image.
			r.deleteArtifactInstance(key)
			if obs.pullFailed {
				upsertImageNotRetrieved(r.Model, artID, obs)
			} else {
				deleteImageNotRetrievedFinding(r.Model, artID)
			}
		}
	}

	r.cleanupStale(desiredKeys, log)
	return ctrl.Result{}, nil
}

func (r *ImageScanReconciler) ensureArtifact(obs *imageObservation) (uuid.UUID, error) {
	// Prefer an existing Artifact matched by hash, then by display name.
	if existing := r.findExistingArtifact(obs); existing != nil {
		id := existing.GetArtifactId()
		r.Index.Put(KindArtifact, obs.key, id)
		return id, nil
	}

	id := artifactIDForImage(obs.key)
	a := artifact.NewArtifact(id)
	a.SetDisplayName(obs.displayName)
	a.SetDescription("Container image observed in the Kubernetes cluster")
	if obs.hash != "" {
		a.SetHash(obs.hash)
	}
	if err := r.Model.AddArtifact(a); err != nil {
		return uuid.Nil, err
	}
	r.Index.Put(KindArtifact, obs.key, id)
	return id, nil
}

func (r *ImageScanReconciler) findExistingArtifact(obs *imageObservation) artifact.Artifact {
	all, err := r.Model.GetArtifacts()
	if err != nil {
		return nil
	}
	if obs.hash != "" {
		for _, a := range all {
			if a.GetHash() == obs.hash {
				return a
			}
		}
	}
	for _, a := range all {
		if a.GetDisplayName() == obs.displayName {
			return a
		}
	}
	// Also match by NameIndex if we previously tracked this key.
	if id := r.Index.Get(KindArtifact, obs.key); id != uuid.Nil {
		return r.Model.GetArtifactById(id)
	}
	return nil
}

func (r *ImageScanReconciler) ensureArtifactInstance(obs *imageObservation, artID uuid.UUID) error {
	instID := artifactInstanceIDForImage(obs.key)
	ai := artifact.NewArtifactInstance(instID)
	ai.SetDisplayName(obs.displayName)
	ai.SetDescription("Retrieved and started container image in this cluster")
	ai.SetArtifactRef(&artifact.ArtifactRef{ArtifactId: artID})
	ai.GetAnnotations().Add(ArtifactInstanceLocationAnnotation, locationAnnotationValue(obs.locations))
	if err := r.Model.AddArtifactInstance(ai); err != nil {
		return err
	}
	r.Index.Put(KindArtifactInstance, obs.key, instID)
	return nil
}

func (r *ImageScanReconciler) deleteArtifactInstance(key string) {
	id := r.Index.Delete(KindArtifactInstance, key)
	if id == uuid.Nil {
		id = artifactInstanceIDForImage(key)
	}
	_ = r.Model.DeleteArtifactInstanceById(id)
}

func (r *ImageScanReconciler) cleanupStale(desiredKeys map[string]struct{}, log interface {
	Error(err error, msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
}) {
	for _, key := range r.Index.Keys(KindArtifactInstance) {
		if _, ok := desiredKeys[key]; ok {
			continue
		}
		id := r.Index.Delete(KindArtifactInstance, key)
		if id == uuid.Nil {
			continue
		}
		if err := r.Model.DeleteArtifactInstanceById(id); err != nil && !errors.Is(err, common.ErrArtifactInstanceNotFound) {
			log.Error(err, "failed to delete stale ArtifactInstance", "key", key)
		}
	}

	for _, key := range r.Index.Keys(KindArtifact) {
		if _, ok := desiredKeys[key]; ok {
			continue
		}
		id := r.Index.Delete(KindArtifact, key)
		if id == uuid.Nil {
			continue
		}
		deleteImageNotRetrievedFinding(r.Model, id)
		// Only delete Artifacts this sensor minted; leave replicated ones alone.
		if isSensorMintedArtifact(id, key) {
			if err := r.Model.DeleteArtifactById(id); err != nil && !errors.Is(err, common.ErrArtifactNotFound) {
				log.Error(err, "failed to delete stale Artifact", "key", key)
			}
		}
	}
}

func upsertImageNotRetrieved(m model.Model, artID uuid.UUID, obs *imageObservation) {
	fID := referenceFindingID(artID, ImageNotRetrieved)
	f := finding.NewFinding(fID)
	f.SetDisplayName(fmt.Sprintf("Image not retrieved: %s", obs.displayName))
	desc := fmt.Sprintf(
		"Container image %s is declared in the cluster but has not been retrieved successfully.",
		obs.displayName,
	)
	if len(obs.failingPods) > 0 {
		desc += " Failing pods: " + strings.Join(obs.failingPods, ", ") + "."
	}
	f.SetDescription(desc)
	f.SetFindingTypeById(finding.TypeIDForKind(ImageNotRetrieved))
	f.SetResources([]*common.ResourceRef{
		{ResourceId: artID, ResourceType: events.ArtifactResource},
	})
	_ = m.AddFinding(f)
}

func deleteImageNotRetrievedFinding(m model.Model, artID uuid.UUID) {
	deleteReferenceFinding(m, artID, ImageNotRetrieved)
}

// RegisterImageFindingTypes registers the well-known FindingType for image
// retrieval findings so it exists even when no findings are active.
func RegisterImageFindingTypes(m model.Model) error {
	id := finding.TypeIDForKind(ImageNotRetrieved)
	ft := finding.NewFindingType(id)
	ft.SetDisplayName("ImageNotRetrieved")
	ft.SetDescription("A container image is declared in the cluster but has not been retrieved successfully.")
	return m.AddFindingType(ft)
}

func (r *ImageScanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mapToClusterScan := handler.EnqueueRequestsFromMapFunc(
		func(_ context.Context, _ client.Object) []reconcile.Request {
			return []reconcile.Request{{NamespacedName: clusterImageScanRequest}}
		},
	)
	return ctrl.NewControllerManagedBy(mgr).
		Named("ImageScan").
		Watches(&corev1.Pod{}, mapToClusterScan).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}
