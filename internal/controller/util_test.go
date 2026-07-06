package controller

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
	"go.emeland.io/modelsrv/pkg/backend"
	"go.emeland.io/modelsrv/pkg/model"
)

func newTestModel(t *testing.T) model.Model {
	t.Helper()
	b, err := backend.New()
	assert.NoError(t, err)
	return b.GetModel()
}

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

func TestResourceID_PrefersSpec(t *testing.T) {
	specID := uuid.New()
	metaID := uuid.New()
	meta := metav1.ObjectMeta{UID: types.UID(metaID.String())}
	assert.Equal(t, specID, resourceID(specID.String(), meta))
}

func TestResourceID_FallsBackToMeta(t *testing.T) {
	metaID := uuid.New()
	meta := metav1.ObjectMeta{UID: types.UID(metaID.String())}
	assert.Equal(t, metaID, resourceID("", meta))
}

// --- annotationUUID ---

func TestAnnotationUUID(t *testing.T) {
	id := uuid.New()
	assert.Equal(t, id, annotationUUID(map[string]string{"key": id.String()}, "key"))
}

func TestAnnotationUUID_Missing(t *testing.T) {
	assert.Equal(t, uuid.Nil, annotationUUID(map[string]string{}, "key"))
}

func TestAnnotationUUID_Empty(t *testing.T) {
	assert.Equal(t, uuid.Nil, annotationUUID(map[string]string{"key": ""}, "key"))
}

func TestAnnotationUUID_Invalid(t *testing.T) {
	assert.Equal(t, uuid.Nil, annotationUUID(map[string]string{"key": "garbage"}, "key"))
}

func TestAnnotationUUID_NilAnnotations(t *testing.T) {
	assert.Equal(t, uuid.Nil, annotationUUID(nil, "key"))
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

	ci, gotID := componentInstanceFromMeta(svc)
	assert.NotNil(t, ci)
	assert.Equal(t, id, gotID)
	assert.Equal(t, "my-deploy", ci.GetDisplayName())
	assert.Equal(t, id, ci.GetInstanceId())
	assert.Equal(t, siID, ci.GetSystemInstance().InstanceId)
	assert.Equal(t, "value", ci.GetAnnotations().GetValue("custom"))
}

func TestComponentInstanceFromMeta_NoUID(t *testing.T) {
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "no-uid"}}
	ci, id := componentInstanceFromMeta(svc)
	assert.Nil(t, ci)
	assert.Equal(t, uuid.Nil, id)
}

func TestComponentInstanceFromMeta_NoSystemInstance(t *testing.T) {
	id := uuid.New()
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "plain",
			UID:  types.UID(id.String()),
		},
	}

	ci, gotID := componentInstanceFromMeta(svc)
	assert.NotNil(t, ci)
	assert.Equal(t, id, gotID)
	assert.Nil(t, ci.GetSystemInstance())
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

	ai, gotID := apiInstanceFromMeta(svc)
	assert.NotNil(t, ai)
	assert.Equal(t, id, gotID)
	assert.Equal(t, "my-svc", ai.GetDisplayName())
	assert.Equal(t, id, ai.GetInstanceId())
	assert.Equal(t, apiID, ai.GetApiRef().ApiID)
	assert.Equal(t, siID, ai.GetSystemInstance().InstanceId)
}

func TestApiInstanceFromMeta_NoUID(t *testing.T) {
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "no-uid"}}
	ai, id := apiInstanceFromMeta(svc)
	assert.Nil(t, ai)
	assert.Equal(t, uuid.Nil, id)
}

func TestApiInstanceFromMeta_NoAnnotations(t *testing.T) {
	id := uuid.New()
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "plain",
			UID:  types.UID(id.String()),
		},
	}

	ai, gotID := apiInstanceFromMeta(svc)
	assert.NotNil(t, ai)
	assert.Equal(t, id, gotID)
	assert.Nil(t, ai.GetApiRef())
	assert.Nil(t, ai.GetSystemInstance())
}

// --- convertSystem ---

func TestConvertSystem_IncludesAbstract(t *testing.T) {
	sysID := uuid.New()
	sys := &v1alpha1.System{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "abstract-sys",
			Namespace: "default",
			UID:       types.UID(uuid.New().String()),
		},
		Spec: v1alpha1.SystemSpec{
			DisplayName: "Abstract System",
			SystemId:    sysID.String(),
			Abstract:    true,
			Version:     v1alpha1.Version{Version: "2.0"},
		},
	}

	obj, id := convertSystem(sys)
	assert.NotNil(t, obj)
	assert.Equal(t, sysID, id)
	assert.True(t, obj.GetAbstract())
}

func TestConvertSystem_NoUUID(t *testing.T) {
	sys := &v1alpha1.System{
		ObjectMeta: metav1.ObjectMeta{Name: "no-id", Namespace: "default"},
		Spec:       v1alpha1.SystemSpec{DisplayName: "x"},
	}
	obj, id := convertSystem(sys)
	assert.Nil(t, obj)
	assert.Equal(t, uuid.Nil, id)
}

// --- convertComponent ---

func TestConvertComponent_ConsumesProvides(t *testing.T) {
	compID := uuid.New()
	apiID := uuid.New()
	comp := &v1alpha1.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "comp",
			Namespace: "default",
			UID:       types.UID(uuid.New().String()),
		},
		Spec: v1alpha1.ComponentSpec{
			DisplayName: "Comp",
			ComponentId: compID.String(),
			Version:     v1alpha1.Version{Version: "1.0"},
			Consumes:    []v1alpha1.APIRef{{ApiId: apiID.String()}},
			Provides:    []v1alpha1.APIRef{{ApiId: apiID.String()}},
		},
	}

	obj, id := convertComponent(comp)
	assert.NotNil(t, obj)
	assert.Equal(t, compID, id)
	assert.Len(t, obj.GetConsumes(), 1)
	assert.Len(t, obj.GetProvides(), 1)
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
	// modelsrv SystemRef is UUID-only; name/version refs are not representable.
	assert.Nil(t, parseSystemRef("", &v1alpha1.VersionRef{Name: "sys", Version: "1.0"}))
}

func TestParseSystemRef_IdTakesPrecedence(t *testing.T) {
	id := uuid.New()
	ref := parseSystemRef(id.String(), &v1alpha1.VersionRef{Name: "sys", Version: "1.0"})
	assert.Equal(t, id, ref.SystemId)
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
