package controller

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
)

// --- uuidFromMeta ---

func TestUuidFromMeta(t *testing.T) {
	id := uuid.New()
	meta := metav1.ObjectMeta{UID: types.UID(id.String())}
	assert.Equal(t, id, uuidFromMeta(meta))
}

func TestUuidFromMeta_Empty(t *testing.T) {
	assert.Equal(t, uuid.Nil, uuidFromMeta(metav1.ObjectMeta{}))
}

func TestUuidFromMeta_Invalid(t *testing.T) {
	meta := metav1.ObjectMeta{UID: "not-a-uuid"}
	assert.Equal(t, uuid.Nil, uuidFromMeta(meta))
}

// --- copyAnnotations ---

func TestCopyAnnotations(t *testing.T) {
	meta := metav1.ObjectMeta{Annotations: map[string]string{"a": "1", "b": "2"}}
	copied := copyAnnotations(meta)
	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, copied)

	// Verify it's a copy, not a reference
	copied["c"] = "3"
	assert.Empty(t, meta.Annotations["c"])
}

func TestCopyAnnotations_Nil(t *testing.T) {
	copied := copyAnnotations(metav1.ObjectMeta{})
	assert.NotNil(t, copied)
	assert.Empty(t, copied)
}

// --- annotationUUID ---

func TestAnnotationUUID(t *testing.T) {
	id := uuid.New()
	meta := metav1.ObjectMeta{Annotations: map[string]string{"key": id.String()}}
	assert.Equal(t, id, annotationUUID(meta, "key"))
}

func TestAnnotationUUID_Missing(t *testing.T) {
	meta := metav1.ObjectMeta{Annotations: map[string]string{}}
	assert.Equal(t, uuid.Nil, annotationUUID(meta, "key"))
}

func TestAnnotationUUID_Empty(t *testing.T) {
	meta := metav1.ObjectMeta{Annotations: map[string]string{"key": ""}}
	assert.Equal(t, uuid.Nil, annotationUUID(meta, "key"))
}

func TestAnnotationUUID_Invalid(t *testing.T) {
	meta := metav1.ObjectMeta{Annotations: map[string]string{"key": "garbage"}}
	assert.Equal(t, uuid.Nil, annotationUUID(meta, "key"))
}

func TestAnnotationUUID_NilAnnotations(t *testing.T) {
	assert.Equal(t, uuid.Nil, annotationUUID(metav1.ObjectMeta{}, "key"))
}

// --- IsOwnedByCronJob ---

func TestIsOwnedByCronJob_True(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "CronJob", Name: "my-cron"},
			},
		},
	}
	assert.True(t, IsOwnedByCronJob(pod))
}

func TestIsOwnedByCronJob_False(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet", Name: "my-rs"},
			},
		},
	}
	assert.False(t, IsOwnedByCronJob(pod))
}

func TestIsOwnedByCronJob_NoOwners(t *testing.T) {
	pod := &corev1.Pod{}
	assert.False(t, IsOwnedByCronJob(pod))
}

func TestIsOwnedByCronJob_MultipleOwners(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet", Name: "my-rs"},
				{Kind: "CronJob", Name: "my-cron"},
			},
		},
	}
	assert.True(t, IsOwnedByCronJob(pod))
}

// --- componentInstanceFromMeta ---

func TestComponentInstanceFromMeta(t *testing.T) {
	id := uuid.New()
	siID := uuid.New()
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-deploy",
			UID:  types.UID(id.String()),
			Annotations: map[string]string{
				AnnotationSystemInstanceID: siID.String(),
				"custom":                   "value",
			},
		},
	}

	ci := componentInstanceFromMeta(svc)
	assert.NotNil(t, ci)
	assert.Equal(t, "my-deploy", ci.DisplayName)
	assert.Equal(t, id, ci.InstanceId)
	assert.Equal(t, siID, ci.SystemInstance.InstanceId)
	assert.Equal(t, "value", ci.Annotations["custom"])
	assert.Equal(t, siID.String(), ci.Annotations[AnnotationSystemInstanceID])
}

func TestComponentInstanceFromMeta_NoUID(t *testing.T) {
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "no-uid"}}
	assert.Nil(t, componentInstanceFromMeta(svc))
}

func TestComponentInstanceFromMeta_NoSystemInstance(t *testing.T) {
	id := uuid.New()
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "plain",
			UID:  types.UID(id.String()),
		},
	}

	ci := componentInstanceFromMeta(svc)
	assert.NotNil(t, ci)
	assert.Nil(t, ci.SystemInstance)
}

// --- apiInstanceFromMeta ---

func TestApiInstanceFromMeta(t *testing.T) {
	id := uuid.New()
	apiID := uuid.New()
	siID := uuid.New()
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-svc",
			UID:  types.UID(id.String()),
			Annotations: map[string]string{
				AnnotationAPIID:            apiID.String(),
				AnnotationSystemInstanceID: siID.String(),
			},
		},
	}

	ai := apiInstanceFromMeta(svc)
	assert.NotNil(t, ai)
	assert.Equal(t, "my-svc", ai.DisplayName)
	assert.Equal(t, id, ai.InstanceId)
	assert.Equal(t, apiID, ai.ApiRef.ApiID)
	assert.Equal(t, siID, ai.SystemInstance.InstanceId)
}

func TestApiInstanceFromMeta_NoUID(t *testing.T) {
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "no-uid"}}
	assert.Nil(t, apiInstanceFromMeta(svc))
}

func TestApiInstanceFromMeta_NoAnnotations(t *testing.T) {
	id := uuid.New()
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "plain",
			UID:  types.UID(id.String()),
		},
	}

	ai := apiInstanceFromMeta(svc)
	assert.NotNil(t, ai)
	assert.Equal(t, uuid.Nil, ai.ApiRef.ApiID)
	assert.Nil(t, ai.SystemInstance)
}

// --- parseVersion ---

func TestParseVersion(t *testing.T) {
	v := parseVersion(v1alpha1.Version{
		Version:        "1.0.0",
		AvailableFrom:  "2023-01-01",
		DeprecatedFrom: "2024-06-15",
		TerminatedFrom: "2025-12-31",
	})

	assert.Equal(t, "1.0.0", v.Version)
	assert.Equal(t, 2023, v.AvailableFrom.Year())
	assert.Equal(t, 6, int(v.DeprecatedFrom.Month()))
	assert.Equal(t, 31, v.TerminatedFrom.Day())
}

func TestParseVersion_Empty(t *testing.T) {
	v := parseVersion(v1alpha1.Version{})
	assert.Empty(t, v.Version)
	assert.Nil(t, v.AvailableFrom)
	assert.Nil(t, v.DeprecatedFrom)
	assert.Nil(t, v.TerminatedFrom)
}

func TestParseDate_Invalid(t *testing.T) {
	assert.Nil(t, parseDate("not-a-date"))
}

// --- parseSystemRef ---

func TestParseSystemRef_ById(t *testing.T) {
	id := uuid.New()
	ref := parseSystemRef(id.String(), nil)
	assert.NotNil(t, ref)
	assert.Equal(t, id, ref.SystemId)
}

func TestParseSystemRef_ByVersionRef(t *testing.T) {
	ref := parseSystemRef("", &v1alpha1.VersionRef{Name: "sys", Version: "1.0"})
	assert.NotNil(t, ref)
	assert.Equal(t, "sys", ref.SystemRef.Name)
	assert.Equal(t, "1.0", ref.SystemRef.Version)
}

func TestParseSystemRef_IdTakesPrecedence(t *testing.T) {
	id := uuid.New()
	ref := parseSystemRef(id.String(), &v1alpha1.VersionRef{Name: "sys", Version: "1.0"})
	assert.Equal(t, id, ref.SystemId)
	assert.Nil(t, ref.SystemRef)
}

func TestParseSystemRef_NeitherSet(t *testing.T) {
	assert.Nil(t, parseSystemRef("", nil))
}

func TestParseSystemRef_InvalidUUID(t *testing.T) {
	assert.Nil(t, parseSystemRef("bad-uuid", nil))
}

func TestParseSystemRef_PartialVersionRef(t *testing.T) {
	assert.Nil(t, parseSystemRef("", &v1alpha1.VersionRef{Name: "sys"}))
	assert.Nil(t, parseSystemRef("", &v1alpha1.VersionRef{Version: "1.0"}))
}
