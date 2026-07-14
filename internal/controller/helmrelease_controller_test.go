package controller

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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

// --- Reconciler integration tests ---

func encodeRelease(t *testing.T, rel *helm.Release) []byte {
	t.Helper()
	data, err := helm.EncodeRelease(rel)
	require.NoError(t, err)
	return data
}

func helmTestFakeClient(t *testing.T, objs ...corev1.Secret) client.Client {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))
	clientObjs := make([]client.Object, len(objs))
	for i := range objs {
		clientObjs[i] = &objs[i]
	}
	return fake.NewClientBuilder().WithScheme(s).WithObjects(clientObjs...).Build()
}

func TestHelmReconcile_CreatesSystemInstance(t *testing.T) {
	ctx := t.Context()
	b, err := backend.New()
	require.NoError(t, err)
	idx := NewNameIndex()

	rel := &helm.Release{
		Name:      "my-app",
		Namespace: "production",
		Version:   1,
		Info:      helm.Info{Status: "deployed"},
		Chart:     helm.Chart{Metadata: helm.ChartMetadata{Name: "my-chart", Version: "1.0.0"}},
		Manifest:  "---\napiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: my-app\n  namespace: production\n",
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sh.helm.release.v1.my-app.v1",
			Namespace: "production",
		},
		Type: "helm.sh/release.v1",
		Data: map[string][]byte{
			"release": encodeRelease(t, rel),
		},
	}

	fakeClient := helmTestFakeClient(t, *secret)
	r := &HelmReleaseReconciler{
		Client: fakeClient,
		Scheme: nil,
		Model:  b.GetModel(),
		Index:  idx,
	}

	nn := types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}
	_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
	require.NoError(t, err)

	// Verify SystemInstance was created with deterministic UUID
	expectedID := uuid.NewSHA1(helmNamespaceUUID, []byte("production/my-app"))
	si := b.GetModel().GetSystemInstanceById(expectedID)
	require.NotNil(t, si)
	assert.Equal(t, "my-app", si.GetDisplayName())
	assert.Equal(t, "my-chart-1.0.0", si.GetAnnotations().GetValue("helm.sh/chart"))
}

func TestHelmReconcile_SkipsNonHelmSecret(t *testing.T) {
	ctx := t.Context()
	b, err := backend.New()
	require.NoError(t, err)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-tls-cert",
			Namespace: "default",
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{"tls.crt": []byte("cert")},
	}

	fakeClient := helmTestFakeClient(t, *secret)
	r := &HelmReleaseReconciler{
		Client: fakeClient,
		Scheme: nil,
		Model:  b.GetModel(),
		Index:  NewNameIndex(),
	}

	nn := types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}
	_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
	require.NoError(t, err)
	// No SystemInstance created
}

func TestHelmReconcile_SkipsIfReleaseDeploisSystemInstance(t *testing.T) {
	ctx := t.Context()
	b, err := backend.New()
	require.NoError(t, err)

	rel := &helm.Release{
		Name:      "self-managed",
		Namespace: "ns",
		Version:   1,
		Info:      helm.Info{Status: "deployed"},
		Chart:     helm.Chart{Metadata: helm.ChartMetadata{Name: "chart", Version: "1.0"}},
		Manifest:  "---\napiVersion: structure.emeland.io/v1alpha1\nkind: SystemInstance\nmetadata:\n  name: my-si\n  namespace: ns\n",
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sh.helm.release.v1.self-managed.v1",
			Namespace: "ns",
		},
		Type: "helm.sh/release.v1",
		Data: map[string][]byte{"release": encodeRelease(t, rel)},
	}

	fakeClient := helmTestFakeClient(t, *secret)
	r := &HelmReleaseReconciler{
		Client: fakeClient,
		Scheme: nil,
		Model:  b.GetModel(),
		Index:  NewNameIndex(),
	}

	nn := types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}
	_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
	require.NoError(t, err)

	expectedID := uuid.NewSHA1(helmNamespaceUUID, []byte("ns/self-managed"))
	assert.Nil(t, b.GetModel().GetSystemInstanceById(expectedID))
}

func TestHelmReconcile_SkipsNonDeployedRelease(t *testing.T) {
	ctx := t.Context()
	b, err := backend.New()
	require.NoError(t, err)

	rel := &helm.Release{
		Name:      "failed-release",
		Namespace: "ns",
		Version:   1,
		Info:      helm.Info{Status: "failed"},
		Chart:     helm.Chart{Metadata: helm.ChartMetadata{Name: "chart", Version: "1.0"}},
		Manifest:  "---\nkind: Deployment\nmetadata:\n  name: app\n  namespace: ns\n",
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sh.helm.release.v1.failed-release.v1",
			Namespace: "ns",
		},
		Type: "helm.sh/release.v1",
		Data: map[string][]byte{"release": encodeRelease(t, rel)},
	}

	fakeClient := helmTestFakeClient(t, *secret)
	r := &HelmReleaseReconciler{
		Client: fakeClient,
		Scheme: nil,
		Model:  b.GetModel(),
		Index:  NewNameIndex(),
	}

	nn := types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}
	_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
	require.NoError(t, err)

	expectedID := uuid.NewSHA1(helmNamespaceUUID, []byte("ns/failed-release"))
	assert.Nil(t, b.GetModel().GetSystemInstanceById(expectedID))
}

func TestHelmReconcile_DeletesSystemInstance(t *testing.T) {
	ctx := t.Context()
	b, err := backend.New()
	require.NoError(t, err)
	idx := NewNameIndex()

	// First: create via reconcile
	rel := &helm.Release{
		Name: "del-me", Namespace: "ns", Version: 2,
		Info:     helm.Info{Status: "deployed"},
		Chart:    helm.Chart{Metadata: helm.ChartMetadata{Name: "c", Version: "1"}},
		Manifest: "---\nkind: Deployment\nmetadata:\n  name: x\n  namespace: ns\n",
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "sh.helm.release.v1.del-me.v2", Namespace: "ns"},
		Type:       "helm.sh/release.v1",
		Data:       map[string][]byte{"release": encodeRelease(t, rel)},
	}

	fakeClient := helmTestFakeClient(t, *secret)
	r := &HelmReleaseReconciler{Client: fakeClient, Scheme: nil, Model: b.GetModel(), Index: idx}

	nn := types.NamespacedName{Name: secret.Name, Namespace: "ns"}
	_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
	require.NoError(t, err)

	expectedID := uuid.NewSHA1(helmNamespaceUUID, []byte("ns/del-me"))
	require.NotNil(t, b.GetModel().GetSystemInstanceById(expectedID))

	// Now simulate delete: reconcile with the secret gone
	fakeClient = helmTestFakeClient(t)
	r.Client = fakeClient
	_, err = r.Reconcile(ctx, reconcile.Request{NamespacedName: nn})
	require.NoError(t, err)

	assert.Nil(t, b.GetModel().GetSystemInstanceById(expectedID))
}
