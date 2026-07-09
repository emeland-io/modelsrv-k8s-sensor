package controller

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/emeland/k8s-model/internal/helm"
	"go.emeland.io/modelsrv/pkg/backend"
	mdlapi "go.emeland.io/modelsrv/pkg/model/api"
	"go.emeland.io/modelsrv/pkg/model/component"
)

func TestCorrelateResources_SetsSystemInstanceRef(t *testing.T) {
	b, err := backend.New()
	require.NoError(t, err)
	m := b.GetModel()
	idx := NewNameIndex()

	// Pre-populate: a ComponentInstance (as if the workload controller created it)
	ciID := uuid.New()
	ci := component.NewComponentInstance(ciID)
	ci.SetDisplayName("my-deploy")
	require.NoError(t, m.AddComponentInstance(ci))
	idx.Put(KindComponentInstance, "production/my-deploy", ciID)

	// The SystemInstance ID
	siID := uuid.New()

	// Simulate manifest resources from the Helm release
	resources := []helm.ManifestResource{
		{Kind: "Deployment", Name: "my-deploy", Namespace: "production"},
		{Kind: "ConfigMap", Name: "my-config", Namespace: "production"}, // not tracked
	}

	r := &HelmReleaseReconciler{Model: m, Index: idx}
	r.correlateResources(resources, "production", siID, logr.Discard())

	// Verify the ComponentInstance now has the SystemInstance ref
	got := m.GetComponentInstanceById(ciID)
	require.NotNil(t, got)
	require.NotNil(t, got.GetSystemInstance())
	assert.Equal(t, siID, got.GetSystemInstance().InstanceId)
}

func TestCorrelateResources_APIInstance(t *testing.T) {
	b, err := backend.New()
	require.NoError(t, err)
	m := b.GetModel()
	idx := NewNameIndex()

	// Pre-populate: an APIInstance (as if the service controller created it)
	aiID := uuid.New()
	ai := mdlapi.NewApiInstance(aiID)
	ai.SetDisplayName("my-svc")
	require.NoError(t, m.AddApiInstance(ai))
	idx.Put(KindAPIInstance, "default/my-svc", aiID)

	siID := uuid.New()

	resources := []helm.ManifestResource{
		{Kind: "Service", Name: "my-svc", Namespace: ""},
	}

	r := &HelmReleaseReconciler{Model: m, Index: idx}
	r.correlateResources(resources, "default", siID, logr.Discard())

	got := m.GetApiInstanceById(aiID)
	require.NotNil(t, got)
	require.NotNil(t, got.GetSystemInstance())
	assert.Equal(t, siID, got.GetSystemInstance().InstanceId)
}

func TestCorrelateResources_MissingResource(t *testing.T) {
	b, err := backend.New()
	require.NoError(t, err)
	m := b.GetModel()
	idx := NewNameIndex()

	siID := uuid.New()

	// Resource not in NameIndex yet -- should not panic
	resources := []helm.ManifestResource{
		{Kind: "Deployment", Name: "not-yet-reconciled", Namespace: "ns"},
	}

	r := &HelmReleaseReconciler{Model: m, Index: idx}
	r.correlateResources(resources, "ns", siID, logr.Discard())
	// No assertion needed -- just verify no panic
}

func TestHelmReleaseReconciler_DeterministicUUID(t *testing.T) {
	id1 := uuid.NewSHA1(helmNamespaceUUID, []byte("production/my-release"))
	id2 := uuid.NewSHA1(helmNamespaceUUID, []byte("production/my-release"))
	assert.Equal(t, id1, id2, "same input should produce same UUID")

	id3 := uuid.NewSHA1(helmNamespaceUUID, []byte("staging/my-release"))
	assert.NotEqual(t, id1, id3, "different namespace should produce different UUID")
}

func TestHelmKindMapping(t *testing.T) {
	assert.Equal(t, KindComponentInstance, helmKindToResourceKind["Deployment"])
	assert.Equal(t, KindComponentInstance, helmKindToResourceKind["StatefulSet"])
	assert.Equal(t, KindComponentInstance, helmKindToResourceKind["DaemonSet"])
	assert.Equal(t, KindComponentInstance, helmKindToResourceKind["Job"])
	assert.Equal(t, KindComponentInstance, helmKindToResourceKind["CronJob"])
	assert.Equal(t, KindAPIInstance, helmKindToResourceKind["Service"])
	assert.Equal(t, KindAPIInstance, helmKindToResourceKind["Ingress"])

	_, ok := helmKindToResourceKind["ConfigMap"]
	assert.False(t, ok)
}
