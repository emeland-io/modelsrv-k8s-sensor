package controller

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
)

// imageArtifactNamespaceUUID is the UUID v5 namespace for sensor-minted Artifact
// and ArtifactInstance IDs derived from container image identity.
var imageArtifactNamespaceUUID = uuid.MustParse("b4c5d6e7-2345-6789-abcd-ef0123456789")

// ArtifactInstanceLocationAnnotation is the modelsrv well-known annotation for
// URLs where a copy of an ArtifactInstance can be found.
const ArtifactInstanceLocationAnnotation = "emeland.io/p8-artifact-instance-location"

// pullFailureReasons are container waiting reasons that indicate the image was
// known (declared in the Pod spec) but not retrieved successfully.
var pullFailureReasons = map[string]struct{}{
	"ErrImagePull":     {},
	"ImagePullBackOff": {},
	"InvalidImageName": {},
}

// sha256DigestRE matches a sha256 digest hex string, optionally prefixed.
var sha256DigestRE = regexp.MustCompile(`(?i)(?:sha256:)?([a-f0-9]{64})`)

// imageObservation aggregates cluster-wide state for one unique container image.
type imageObservation struct {
	// key is the identity used for UUID derivation and NameIndex (digest or ref).
	key string
	// displayName is the human-readable image reference (prefer name@sha256:…).
	displayName string
	// hash is modelsrv form "SHA256:<hex>" when a digest is known; empty otherwise.
	hash string
	// started is true when at least one container of this image is Running or Terminated with imageID.
	started bool
	// pullFailed is true when at least one container is waiting on a pull failure and none started.
	pullFailed bool
	// locations are oci:// refs and/or node names where a started copy was seen.
	locations []string
	// failingPods are "namespace/name" of pods with pull failures (for finding text).
	failingPods []string
}

// imageIdentity holds the normalized identity fields for a container image.
type imageIdentity struct {
	key         string
	displayName string
	hash        string // SHA256:<hex> or empty
}

// parseImageID extracts a SHA256 hash from a CRI imageID string.
// Accepts forms such as:
//   - docker-pullable://repo/name@sha256:abc…
//   - containerd://sha256:abc…
//   - sha256:abc…
//
// Returns modelsrv form "SHA256:<hex>" or empty if no digest is found.
func parseImageID(imageID string) string {
	if imageID == "" {
		return ""
	}
	m := sha256DigestRE.FindStringSubmatch(imageID)
	if m == nil {
		return ""
	}
	return "SHA256:" + strings.ToLower(m[1])
}

// normalizeImageRef returns a trimmed image reference suitable as a display name.
func normalizeImageRef(ref string) string {
	return strings.TrimSpace(ref)
}

// identityFromImage builds an imageIdentity from a declared image ref and optional CRI imageID.
// Prefer digest from imageID; fall back to a digest embedded in the ref (name@sha256:…);
// otherwise use the normalized ref as the key.
func identityFromImage(imageRef, imageID string) imageIdentity {
	ref := normalizeImageRef(imageRef)
	hash := parseImageID(imageID)
	if hash == "" {
		// Try digest in the image reference itself (e.g. nginx@sha256:…).
		hash = parseImageID(ref)
	}

	if hash != "" {
		hex := strings.TrimPrefix(hash, "SHA256:")
		return imageIdentity{
			key:         "sha256:" + hex,
			displayName: displayNameForDigest(ref, hex),
			hash:        hash,
		}
	}

	return imageIdentity{
		key:         ref,
		displayName: ref,
		hash:        "",
	}
}

// displayNameForDigest builds a stable display name from an image ref and digest hex.
func displayNameForDigest(ref, hex string) string {
	if at := strings.LastIndex(ref, "@"); at >= 0 {
		return ref[:at] + "@sha256:" + hex
	}
	if ref != "" {
		base := ref
		if i := strings.LastIndex(ref, ":"); i > strings.LastIndex(ref, "/") {
			base = ref[:i]
		}
		return base + "@sha256:" + hex
	}
	return "sha256:" + hex
}

// artifactIDForImage returns the deterministic UUID for a sensor-minted Artifact.
func artifactIDForImage(key string) uuid.UUID {
	return uuid.NewSHA1(imageArtifactNamespaceUUID, []byte("artifact:"+key))
}

// artifactInstanceIDForImage returns the deterministic UUID for a sensor-minted ArtifactInstance.
func artifactInstanceIDForImage(key string) uuid.UUID {
	return uuid.NewSHA1(imageArtifactNamespaceUUID, []byte("instance:"+key))
}

// isSensorMintedArtifact reports whether id was derived from this sensor's image namespace
// for the given key (used to decide whether the sensor may delete the Artifact).
func isSensorMintedArtifact(id uuid.UUID, key string) bool {
	return id == artifactIDForImage(key)
}

// containerStarted reports whether a ContainerStatus represents a retrieved-and-started image.
func containerStarted(cs corev1.ContainerStatus) bool {
	if cs.ImageID == "" {
		return false
	}
	return cs.State.Running != nil || cs.State.Terminated != nil
}

// containerPullFailed reports whether a ContainerStatus is waiting on a pull failure.
func containerPullFailed(cs corev1.ContainerStatus) bool {
	if cs.State.Waiting == nil {
		return false
	}
	_, ok := pullFailureReasons[cs.State.Waiting.Reason]
	return ok
}

// collectImageObservations walks all Pods and aggregates per-image state.
func collectImageObservations(pods []corev1.Pod) map[string]*imageObservation {
	obs := make(map[string]*imageObservation)
	for i := range pods {
		observePodImages(obs, &pods[i])
	}
	// Pull failure only counts when the image never started anywhere.
	for _, o := range obs {
		if o.started {
			o.pullFailed = false
			o.failingPods = nil
		}
	}
	return obs
}

func ensureObservation(obs map[string]*imageObservation, id imageIdentity) *imageObservation {
	o, ok := obs[id.key]
	if !ok {
		o = &imageObservation{
			key:         id.key,
			displayName: id.displayName,
			hash:        id.hash,
		}
		obs[id.key] = o
		return o
	}
	// Prefer a hash / digest-qualified display name when we learn one later.
	if o.hash == "" && id.hash != "" {
		o.hash = id.hash
		o.displayName = id.displayName
	}
	return o
}

func addUnique(list *[]string, value string) {
	if value == "" {
		return
	}
	for _, existing := range *list {
		if existing == value {
			return
		}
	}
	*list = append(*list, value)
}

func mergeTagObservation(obs map[string]*imageObservation, digestObs *imageObservation, tagKey string) {
	if tagKey == "" || tagKey == digestObs.key {
		return
	}
	prev, ok := obs[tagKey]
	if !ok {
		return
	}
	if prev.started {
		digestObs.started = true
	}
	if prev.pullFailed {
		digestObs.pullFailed = true
	}
	for _, loc := range prev.locations {
		addUnique(&digestObs.locations, loc)
	}
	for _, p := range prev.failingPods {
		addUnique(&digestObs.failingPods, p)
	}
	delete(obs, tagKey)
}

func registerDeclaredImages(obs map[string]*imageObservation, images []string) {
	for _, image := range images {
		if image == "" {
			continue
		}
		_ = ensureObservation(obs, identityFromImage(image, ""))
	}
}

func observeContainerStatus(obs map[string]*imageObservation, pod *corev1.Pod, podKey string, cs corev1.ContainerStatus) {
	id := identityFromImage(cs.Image, cs.ImageID)
	if id.key == "" {
		return
	}
	o := ensureObservation(obs, id)
	if id.hash != "" {
		mergeTagObservation(obs, o, normalizeImageRef(cs.Image))
	}
	if containerStarted(cs) {
		o.started = true
		addUnique(&o.locations, "oci://"+o.displayName)
		if pod.Spec.NodeName != "" {
			addUnique(&o.locations, "node://"+pod.Spec.NodeName)
		}
	}
	if containerPullFailed(cs) {
		o.pullFailed = true
		addUnique(&o.failingPods, podKey)
	}
}

func observePodImages(obs map[string]*imageObservation, pod *corev1.Pod) {
	podKey := pod.Namespace + "/" + pod.Name

	declared := make([]string, 0, len(pod.Spec.Containers)+len(pod.Spec.InitContainers)+len(pod.Spec.EphemeralContainers))
	for _, c := range pod.Spec.Containers {
		declared = append(declared, c.Image)
	}
	for _, c := range pod.Spec.InitContainers {
		declared = append(declared, c.Image)
	}
	for _, c := range pod.Spec.EphemeralContainers {
		declared = append(declared, c.Image)
	}
	registerDeclaredImages(obs, declared)

	for _, cs := range pod.Status.ContainerStatuses {
		observeContainerStatus(obs, pod, podKey, cs)
	}
	for _, cs := range pod.Status.InitContainerStatuses {
		observeContainerStatus(obs, pod, podKey, cs)
	}
	for _, cs := range pod.Status.EphemeralContainerStatuses {
		observeContainerStatus(obs, pod, podKey, cs)
	}
}

// locationAnnotationValue encodes locations as a JSON list for the well-known annotation.
func locationAnnotationValue(locations []string) string {
	if len(locations) == 0 {
		return "[]"
	}
	b, err := json.Marshal(locations)
	if err != nil {
		return fmt.Sprintf("%q", locations[0])
	}
	return string(b)
}
