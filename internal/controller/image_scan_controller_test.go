package controller

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"go.emeland.io/modelsrv/pkg/backend"
	"go.emeland.io/modelsrv/pkg/model/artifact"
	"go.emeland.io/modelsrv/pkg/model/finding"
)

var _ = Describe("ImageScanReconciler", func() {
	var (
		ctx    = context.Background()
		digest = "9e9b755d63b36acf30c12a9a3fc379243714c1c6d3dd72861da637f336ebb35b"
	)

	reconcileCluster := func(r *ImageScanReconciler) {
		_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: clusterImageScanRequest})
		Expect(err).NotTo(HaveOccurred())
	}

	runningPod := func(name, ns, image, imageID string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				UID:       types.UID(uuid.New().String()),
			},
			Spec: corev1.PodSpec{
				NodeName: "worker-1",
				Containers: []corev1.Container{{
					Name:  "main",
					Image: image,
				}},
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{{
					Name:    "main",
					Image:   image,
					ImageID: imageID,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				}},
			},
		}
	}

	pullFailPod := func(name, ns, image string) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				UID:       types.UID(uuid.New().String()),
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "main",
					Image: image,
				}},
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{{
					Name:  "main",
					Image: image,
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"},
					},
				}},
			},
		}
	}

	It("creates Artifact and ArtifactInstance for a running Pod with digest", func() {
		pod := runningPod("app", "default", "nginx:1.25", "docker-pullable://nginx@sha256:"+digest)
		b, err := backend.New()
		Expect(err).NotTo(HaveOccurred())
		Expect(RegisterImageFindingTypes(b.GetModel())).To(Succeed())
		idx := NewNameIndex()
		r := &ImageScanReconciler{
			Client: newFakeClient(pod),
			Scheme: testScheme,
			Model:  b.GetModel(),
			Index:  idx,
		}

		reconcileCluster(r)

		key := "sha256:" + digest
		artID := artifactIDForImage(key)
		art := b.GetModel().GetArtifactById(artID)
		Expect(art).NotTo(BeNil())
		Expect(art.GetHash()).To(Equal("SHA256:" + digest))
		Expect(art.GetDisplayName()).To(Equal("nginx@sha256:" + digest))

		instID := artifactInstanceIDForImage(key)
		inst := b.GetModel().GetArtifactInstanceById(instID)
		Expect(inst).NotTo(BeNil())
		Expect(inst.GetArtifactRef()).NotTo(BeNil())
		Expect(inst.GetArtifactRef().ArtifactId).To(Equal(artID))
		Expect(inst.GetAnnotations().GetValue(ArtifactInstanceLocationAnnotation)).To(ContainSubstring("oci://"))

		fID := referenceFindingID(artID, ImageNotRetrieved)
		Expect(b.GetModel().GetFindingById(fID)).To(BeNil())
	})

	It("deduplicates the same digest across two Pods into one Artifact and Instance", func() {
		pod1 := runningPod("app-a", "default", "nginx:1.25", "docker-pullable://nginx@sha256:"+digest)
		pod2 := runningPod("app-b", "other", "nginx:1.26", "containerd://sha256:"+digest)
		b, err := backend.New()
		Expect(err).NotTo(HaveOccurred())
		idx := NewNameIndex()
		r := &ImageScanReconciler{
			Client: newFakeClient(pod1, pod2),
			Scheme: testScheme,
			Model:  b.GetModel(),
			Index:  idx,
		}

		reconcileCluster(r)

		arts, err := b.GetModel().GetArtifacts()
		Expect(err).NotTo(HaveOccurred())
		Expect(arts).To(HaveLen(1))

		insts, err := b.GetModel().GetArtifactInstances()
		Expect(err).NotTo(HaveOccurred())
		Expect(insts).To(HaveLen(1))
	})

	It("reuses an existing Artifact with matching hash instead of creating a duplicate", func() {
		existingID := uuid.New()
		existing := artifact.NewArtifact(existingID)
		existing.SetDisplayName("catalog-nginx")
		existing.SetHash("SHA256:" + digest)

		pod := runningPod("app", "default", "library/nginx:1.25", "docker-pullable://nginx@sha256:"+digest)
		b, err := backend.New()
		Expect(err).NotTo(HaveOccurred())
		Expect(b.GetModel().AddArtifact(existing)).To(Succeed())

		r := &ImageScanReconciler{
			Client: newFakeClient(pod),
			Scheme: testScheme,
			Model:  b.GetModel(),
			Index:  NewNameIndex(),
		}
		reconcileCluster(r)

		arts, err := b.GetModel().GetArtifacts()
		Expect(err).NotTo(HaveOccurred())
		Expect(arts).To(HaveLen(1))
		Expect(arts[0].GetArtifactId()).To(Equal(existingID))

		insts, err := b.GetModel().GetArtifactInstances()
		Expect(err).NotTo(HaveOccurred())
		Expect(insts).To(HaveLen(1))
		Expect(insts[0].GetArtifactRef().ArtifactId).To(Equal(existingID))

		// Sensor must not delete a non-minted Artifact on cleanup of other images.
		Expect(isSensorMintedArtifact(existingID, "sha256:"+digest)).To(BeFalse())
	})

	It("emits ImageNotRetrieved for ImagePullBackOff without creating an ArtifactInstance", func() {
		pod := pullFailPod("broken", "default", "missing:latest")
		b, err := backend.New()
		Expect(err).NotTo(HaveOccurred())
		Expect(RegisterImageFindingTypes(b.GetModel())).To(Succeed())

		r := &ImageScanReconciler{
			Client: newFakeClient(pod),
			Scheme: testScheme,
			Model:  b.GetModel(),
			Index:  NewNameIndex(),
		}
		reconcileCluster(r)

		key := "missing:latest"
		artID := artifactIDForImage(key)
		Expect(b.GetModel().GetArtifactById(artID)).NotTo(BeNil())

		insts, err := b.GetModel().GetArtifactInstances()
		Expect(err).NotTo(HaveOccurred())
		Expect(insts).To(BeEmpty())

		f := b.GetModel().GetFindingById(referenceFindingID(artID, ImageNotRetrieved))
		Expect(f).NotTo(BeNil())
		Expect(f.GetFindingTypeId()).To(Equal(finding.TypeIDForKind(ImageNotRetrieved)))
		Expect(f.GetDescription()).To(ContainSubstring("default/broken"))
	})

	It("creates ArtifactInstance and clears finding when pull recovers to Running", func() {
		failing := pullFailPod("app", "default", "nginx:1.25")
		b, err := backend.New()
		Expect(err).NotTo(HaveOccurred())
		Expect(RegisterImageFindingTypes(b.GetModel())).To(Succeed())
		idx := NewNameIndex()

		r := &ImageScanReconciler{
			Client: newFakeClient(failing),
			Scheme: testScheme,
			Model:  b.GetModel(),
			Index:  idx,
		}
		reconcileCluster(r)

		failArtID := artifactIDForImage("nginx:1.25")
		Expect(b.GetModel().GetFindingById(referenceFindingID(failArtID, ImageNotRetrieved))).NotTo(BeNil())

		// Simulate recovery: replace the pod with a running one that has a digest.
		running := runningPod("app", "default", "nginx:stable", "docker-pullable://nginx@sha256:"+digest)
		r.Client = newFakeClient(running)
		reconcileCluster(r)

		key := "sha256:" + digest
		artID := idx.Get(KindArtifact, key)
		Expect(artID).NotTo(Equal(uuid.Nil))
		Expect(b.GetModel().GetArtifactInstanceById(artifactInstanceIDForImage(key))).NotTo(BeNil())
		Expect(b.GetModel().GetFindingById(referenceFindingID(artID, ImageNotRetrieved))).To(BeNil())
		// Tag-only finding from the earlier failure should also be gone after cleanup.
		Expect(b.GetModel().GetFindingById(referenceFindingID(failArtID, ImageNotRetrieved))).To(BeNil())
	})

	It("removes ArtifactInstance and sensor-minted Artifact when Pods disappear", func() {
		pod := runningPod("app", "default", "nginx:latest", "docker-pullable://nginx@sha256:"+digest)
		b, err := backend.New()
		Expect(err).NotTo(HaveOccurred())
		idx := NewNameIndex()
		r := &ImageScanReconciler{
			Client: newFakeClient(pod),
			Scheme: testScheme,
			Model:  b.GetModel(),
			Index:  idx,
		}
		reconcileCluster(r)

		key := "sha256:" + digest
		Expect(b.GetModel().GetArtifactById(artifactIDForImage(key))).NotTo(BeNil())
		Expect(b.GetModel().GetArtifactInstanceById(artifactInstanceIDForImage(key))).NotTo(BeNil())

		r.Client = newFakeClient() // all pods gone
		reconcileCluster(r)

		Expect(b.GetModel().GetArtifactInstanceById(artifactInstanceIDForImage(key))).To(BeNil())
		Expect(b.GetModel().GetArtifactById(artifactIDForImage(key))).To(BeNil())
		Expect(idx.Keys(KindArtifact)).To(BeEmpty())
		Expect(idx.Keys(KindArtifactInstance)).To(BeEmpty())
	})
})
