package crdcheck_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/finding"

	"gitlab.com/emeland/k8s-model/internal/crdcheck"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

func fakeDiscoveryWithResources(resources []*metav1.APIResourceList) discovery.DiscoveryInterface {
	cs := fakeclientset.NewSimpleClientset()
	fakeDisc := cs.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDisc.Fake.Resources = resources
	return fakeDisc
}

// partialFailDiscovery wraps a real fake discovery client but overrides
// ServerGroupsAndResources to return both partial data and an error.
type partialFailDiscovery struct {
	*fakediscovery.FakeDiscovery
	resources []*metav1.APIResourceList
	err       error
}

func (d *partialFailDiscovery) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	return nil, d.resources, d.err
}

func newPartialFailDiscovery(resources []*metav1.APIResourceList, err error) discovery.DiscoveryInterface {
	cs := fakeclientset.NewSimpleClientset()
	fakeDisc := cs.Discovery().(*fakediscovery.FakeDiscovery)
	return &partialFailDiscovery{
		FakeDiscovery: fakeDisc,
		resources:     resources,
		err:           err,
	}
}

func TestCheck_AllAvailable(t *testing.T) {
	checklist := []crdcheck.CRDEntry{
		{Group: "cert-manager.io", Version: "v1", Resource: "certificates", DisplayName: "Certificate", Category: "CertManager"},
	}
	resources := []*metav1.APIResourceList{
		{
			GroupVersion: "cert-manager.io/v1",
			APIResources: []metav1.APIResource{
				{Name: "certificates"},
			},
		},
	}
	disc := fakeDiscoveryWithResources(resources)
	result := crdcheck.Check(context.Background(), disc, checklist)

	assert.Len(t, result.Available, 1)
	assert.Empty(t, result.Missing)
}

func TestCheck_SomeMissing(t *testing.T) {
	checklist := []crdcheck.CRDEntry{
		{Group: "cert-manager.io", Version: "v1", Resource: "certificates", DisplayName: "Certificate", Category: "CertManager"},
		{Group: "monitoring.coreos.com", Version: "v1", Resource: "servicemonitors", DisplayName: "ServiceMonitor", Category: "Prometheus"},
	}
	resources := []*metav1.APIResourceList{
		{
			GroupVersion: "cert-manager.io/v1",
			APIResources: []metav1.APIResource{
				{Name: "certificates"},
			},
		},
	}
	disc := fakeDiscoveryWithResources(resources)
	result := crdcheck.Check(context.Background(), disc, checklist)

	assert.Len(t, result.Available, 1)
	assert.Len(t, result.Missing, 1)
	assert.Equal(t, "ServiceMonitor", result.Missing[0].DisplayName)
}

func TestCheck_AllMissing(t *testing.T) {
	checklist := []crdcheck.CRDEntry{
		{Group: "cert-manager.io", Version: "v1", Resource: "certificates", DisplayName: "Certificate", Category: "CertManager"},
	}
	disc := fakeDiscoveryWithResources(nil)
	result := crdcheck.Check(context.Background(), disc, checklist)

	assert.Empty(t, result.Available)
	assert.Len(t, result.Missing, 1)
}

func TestCheck_PartialDiscoveryFailure(t *testing.T) {
	// Simulates the common case where some API groups fail (e.g. metrics-server
	// is down) but other groups return valid data. ServerGroupsAndResources
	// returns both an error and partial resource lists.
	checklist := []crdcheck.CRDEntry{
		{Group: "cert-manager.io", Version: "v1", Resource: "certificates", DisplayName: "Certificate", Category: "CertManager"},
		{Group: "monitoring.coreos.com", Version: "v1", Resource: "servicemonitors", DisplayName: "ServiceMonitor", Category: "Prometheus"},
	}

	// Only cert-manager resources are returned; monitoring group failed.
	partialResources := []*metav1.APIResourceList{
		{
			GroupVersion: "cert-manager.io/v1",
			APIResources: []metav1.APIResource{
				{Name: "certificates"},
			},
		},
	}

	disc := newPartialFailDiscovery(partialResources, &discovery.ErrGroupDiscoveryFailed{
		Groups: map[schema.GroupVersion]error{
			{Group: "monitoring.coreos.com", Version: "v1"}: errors.New("connection refused"),
		},
	})

	result := crdcheck.Check(context.Background(), disc, checklist)

	// Should use partial data: cert-manager found, monitoring missing.
	assert.Len(t, result.Available, 1)
	assert.Equal(t, "Certificate", result.Available[0].DisplayName)
	assert.Len(t, result.Missing, 1)
	assert.Equal(t, "ServiceMonitor", result.Missing[0].DisplayName)
}

func TestCheck_TotalDiscoveryFailure(t *testing.T) {
	// When resourceLists is nil (total failure), all CRDs are reported missing.
	checklist := []crdcheck.CRDEntry{
		{Group: "cert-manager.io", Version: "v1", Resource: "certificates", DisplayName: "Certificate", Category: "CertManager"},
	}

	disc := newPartialFailDiscovery(nil, errors.New("connection refused"))

	result := crdcheck.Check(context.Background(), disc, checklist)

	assert.Empty(t, result.Available)
	assert.Len(t, result.Missing, 1)
}

func TestLogAndReport_CreatesFindingsForMissing(t *testing.T) {
	sink := events.NewDummySink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	result := crdcheck.CheckResult{
		Available: []crdcheck.CRDEntry{
			{Group: "cert-manager.io", Version: "v1", Resource: "certificates", DisplayName: "Certificate", Category: "CertManager"},
		},
		Missing: []crdcheck.CRDEntry{
			{Group: "monitoring.coreos.com", Version: "v1", Resource: "servicemonitors", DisplayName: "ServiceMonitor", Category: "Prometheus"},
		},
	}

	crdcheck.LogAndReport(logr.Discard(), m, result)

	// Should have the finding type registered.
	typeID := finding.TypeIDForKind("CRDNotAvailable")
	ft := m.GetFindingTypeById(typeID)
	require.NotNil(t, ft)
	assert.Equal(t, "CRDNotAvailable", ft.GetDisplayName())

	// Should have one finding.
	findings, err := m.GetFindings()
	require.NoError(t, err)
	assert.Len(t, findings, 1)
	assert.Contains(t, findings[0].GetDisplayName(), "ServiceMonitor")
}

func TestLogAndReport_NoFindingsWhenAllAvailable(t *testing.T) {
	sink := events.NewDummySink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	result := crdcheck.CheckResult{
		Available: []crdcheck.CRDEntry{
			{Group: "cert-manager.io", Version: "v1", Resource: "certificates", DisplayName: "Certificate", Category: "CertManager"},
		},
	}

	crdcheck.LogAndReport(logr.Discard(), m, result)

	findings, err := m.GetFindings()
	require.NoError(t, err)
	assert.Empty(t, findings)
}

func TestLogAndReport_DeterministicFindingID(t *testing.T) {
	sink := events.NewDummySink()
	m, err := model.NewModel(sink)
	require.NoError(t, err)

	result := crdcheck.CheckResult{
		Missing: []crdcheck.CRDEntry{
			{Group: "monitoring.coreos.com", Version: "v1", Resource: "servicemonitors", DisplayName: "ServiceMonitor", Category: "Prometheus"},
		},
	}

	// Run twice - should upsert, not duplicate.
	crdcheck.LogAndReport(logr.Discard(), m, result)
	crdcheck.LogAndReport(logr.Discard(), m, result)

	findings, err := m.GetFindings()
	require.NoError(t, err)
	assert.Len(t, findings, 1)
}

func TestParseChecklist_Valid(t *testing.T) {
	input := "cert-manager.io/v1/certificates,monitoring.coreos.com/v1/servicemonitors"
	entries, err := crdcheck.ParseChecklist(input)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	// First entry should be enriched from DefaultChecklist.
	assert.Equal(t, "Certificate", entries[0].DisplayName)
	assert.Equal(t, "CertManager", entries[0].Category)

	// Second entry also known.
	assert.Equal(t, "ServiceMonitor", entries[1].DisplayName)
}

func TestParseChecklist_Unknown(t *testing.T) {
	input := "custom.io/v1/widgets"
	entries, err := crdcheck.ParseChecklist(input)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "widgets", entries[0].DisplayName)
	assert.Equal(t, "custom.io", entries[0].Category)
}

func TestParseChecklist_Empty(t *testing.T) {
	entries, err := crdcheck.ParseChecklist("")
	require.NoError(t, err)
	assert.Nil(t, entries)
}

func TestParseChecklist_Invalid(t *testing.T) {
	_, err := crdcheck.ParseChecklist("invalid-no-slashes")
	assert.Error(t, err)

	_, err = crdcheck.ParseChecklist("only/one")
	assert.Error(t, err)

	_, err = crdcheck.ParseChecklist("too/many/slashes/here")
	assert.Error(t, err)
}

func TestParseChecklist_WhitespaceHandling(t *testing.T) {
	input := " cert-manager.io/v1/certificates , monitoring.coreos.com/v1/servicemonitors "
	entries, err := crdcheck.ParseChecklist(input)
	require.NoError(t, err)
	require.Len(t, entries, 2)
}
