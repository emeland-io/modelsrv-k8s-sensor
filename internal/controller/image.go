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
	"ErrImagePull":      {},
	"ImagePullBackOff":  {},
	"InvalidImageName":  {},
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
		display := ref
		if at := strings.LastIndex(ref, "@"); at >= 0 {
			display = ref[:at] + "@sha256:" + hex
		} else if ref != "" {
			// Strip tag if present and attach digest for a stable display name.
			base := ref
			if i := strings.LastIndex(ref, ":"); i > strings.LastIndex(ref, "/") {
				base = ref[:i]
			}
			display = base + "@sha256:" + hex
		} else {
			display = "sha256:" + hex
		}
		return imageIdentity{
			key:         "sha256:" + hex,
			displayName: display,
			hash:        hash,
		}
	}

	return imageIdentity{
		key:         ref,
		displayName: ref,
		hash:        "",
	}
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

	ensure := func(id imageIdentity) *imageObservation {
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

	addLocation := func(o *imageObservation, loc string) {
		if loc == "" {
			return
		}
		for _, existing := range o.locations {
			if existing == loc {
				return
			}
		}
		o.locations = append(o.locations, loc)
	}

	addFailingPod := func(o *imageObservation, podKey string) {
		for _, existing := range o.failingPods {
			if existing == podKey {
				return
			}
		}
		o.failingPods = append(o.failingPods, podKey)
	}

	for i := range pods {
		pod := &pods[i]
		podKey := pod.Namespace + "/" + pod.Name

		// Spec: register declared images as known.
		for _, c := range pod.Spec.Containers {
			if c.Image == "" {
				continue
			}
			_ = ensure(identityFromImage(c.Image, ""))
		}
		for _, c := range pod.Spec.InitContainers {
			if c.Image == "" {
				continue
			}
			_ = ensure(identityFromImage(c.Image, ""))
		}
		for _, c := range pod.Spec.EphemeralContainers {
			if c.Image == "" {
				continue
			}
			_ = ensure(identityFromImage(c.Image, ""))
		}

		processStatus := func(cs corev1.ContainerStatus) {
			id := identityFromImage(cs.Image, cs.ImageID)
			if id.key == "" {
				return
			}
			o := ensure(id)
			// When status reveals a digest, fold any earlier tag-only observation
			// (keyed by the declared image ref) into this digest identity.
			if id.hash != "" {
				tagKey := normalizeImageRef(cs.Image)
				if tagKey != "" && tagKey != id.key {
					if prev, ok := obs[tagKey]; ok {
						if prev.started {
							o.started = true
						}
						if prev.pullFailed {
							o.pullFailed = true
						}
						for _, loc := range prev.locations {
							addLocation(o, loc)
						}
						for _, p := range prev.failingPods {
							addFailingPod(o, p)
						}
						delete(obs, tagKey)
					}
				}
			}
			if containerStarted(cs) {
				o.started = true
				addLocation(o, "oci://"+o.displayName)
				if pod.Spec.NodeName != "" {
					addLocation(o, "node://"+pod.Spec.NodeName)
				}
			}
			if containerPullFailed(cs) {
				o.pullFailed = true
				addFailingPod(o, podKey)
			}
		}

		for _, cs := range pod.Status.ContainerStatuses {
			processStatus(cs)
		}
		for _, cs := range pod.Status.InitContainerStatuses {
			processStatus(cs)
		}
		for _, cs := range pod.Status.EphemeralContainerStatuses {
			processStatus(cs)
		}
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
