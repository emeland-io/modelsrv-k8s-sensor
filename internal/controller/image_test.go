package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseImageID(t *testing.T) {
	cases := []struct {
		name    string
		imageID string
		want    string
	}{
		{
			name:    "docker-pullable with digest",
			imageID: "docker-pullable://nginx@sha256:9e9b755d63b36acf30c12a9a3fc379243714c1c6d3dd72861da637f336ebb35b",
			want:    "SHA256:9e9b755d63b36acf30c12a9a3fc379243714c1c6d3dd72861da637f336ebb35b",
		},
		{
			name:    "containerd sha256",
			imageID: "containerd://sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			want:    "SHA256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		{
			name:    "bare sha256",
			imageID: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			want:    "SHA256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		{
			name:    "uppercase hex normalized",
			imageID: "sha256:CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC",
			want:    "SHA256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		},
		{
			name:    "empty",
			imageID: "",
			want:    "",
		},
		{
			name:    "no digest",
			imageID: "docker://nginx:latest",
			want:    "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, parseImageID(tc.imageID))
		})
	}
}

func TestIdentityFromImage(t *testing.T) {
	digest := "9e9b755d63b36acf30c12a9a3fc379243714c1c6d3dd72861da637f336ebb35b"
	id := identityFromImage("nginx:1.25", "docker-pullable://nginx@sha256:"+digest)
	assert.Equal(t, "sha256:"+digest, id.key)
	assert.Equal(t, "SHA256:"+digest, id.hash)
	assert.Equal(t, "nginx@sha256:"+digest, id.displayName)

	id = identityFromImage("nginx:1.25", "")
	assert.Equal(t, "nginx:1.25", id.key)
	assert.Empty(t, id.hash)
	assert.Equal(t, "nginx:1.25", id.displayName)

	id = identityFromImage("nginx@sha256:"+digest, "")
	assert.Equal(t, "sha256:"+digest, id.key)
	assert.Equal(t, "SHA256:"+digest, id.hash)
}

func TestArtifactIDsDeterministic(t *testing.T) {
	a1 := artifactIDForImage("sha256:abc")
	a2 := artifactIDForImage("sha256:abc")
	assert.Equal(t, a1, a2)
	assert.NotEqual(t, a1, artifactIDForImage("sha256:def"))
	assert.NotEqual(t, a1, artifactInstanceIDForImage("sha256:abc"))
	assert.True(t, isSensorMintedArtifact(a1, "sha256:abc"))
	assert.False(t, isSensorMintedArtifact(a1, "other-key"))
}

func TestCollectImageObservationsStarted(t *testing.T) {
	digest := "9e9b755d63b36acf30c12a9a3fc379243714c1c6d3dd72861da637f336ebb35b"
	pods := []corev1.Pod{{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
			Containers: []corev1.Container{{
				Name:  "c",
				Image: "nginx:1.25",
			}},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:    "c",
				Image:   "nginx:1.25",
				ImageID: "docker-pullable://nginx@sha256:" + digest,
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{},
				},
			}},
		},
	}}

	obs := collectImageObservations(pods)
	require.Len(t, obs, 1)
	o := obs["sha256:"+digest]
	require.NotNil(t, o)
	assert.True(t, o.started)
	assert.False(t, o.pullFailed)
	assert.Equal(t, "SHA256:"+digest, o.hash)
	assert.Contains(t, o.locations, "node://node-1")
}

func TestCollectImageObservationsPullFailed(t *testing.T) {
	pods := []corev1.Pod{{
		ObjectMeta: metav1.ObjectMeta{Name: "bad", Namespace: "ns"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "c",
				Image: "missing:latest",
			}},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:  "c",
				Image: "missing:latest",
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"},
				},
			}},
		},
	}}

	obs := collectImageObservations(pods)
	require.Len(t, obs, 1)
	o := obs["missing:latest"]
	require.NotNil(t, o)
	assert.False(t, o.started)
	assert.True(t, o.pullFailed)
	assert.Equal(t, []string{"ns/bad"}, o.failingPods)
}

func TestCollectImageObservationsMergesTagIntoDigest(t *testing.T) {
	digest := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	pods := []corev1.Pod{{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Image: "busybox:1.36"}},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{{
				Image:   "busybox:1.36",
				ImageID: "containerd://sha256:" + digest,
				State:   corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
			}},
		},
	}}

	obs := collectImageObservations(pods)
	require.Len(t, obs, 1)
	_, hasTag := obs["busybox:1.36"]
	assert.False(t, hasTag)
	assert.NotNil(t, obs["sha256:"+digest])
}

func TestLocationAnnotationValue(t *testing.T) {
	assert.Equal(t, "[]", locationAnnotationValue(nil))
	assert.Equal(t, `["oci://nginx"]`, locationAnnotationValue([]string{"oci://nginx"}))
}
