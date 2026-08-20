package crdcheck_test

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/finding"

	"gitlab.com/emeland/k8s-model/internal/crdcheck"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
