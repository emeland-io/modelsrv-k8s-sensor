package sensor_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.emeland.io/modelsrv/pkg/backend"

	"gitlab.com/emeland/k8s-model/internal/sensor"
)

func TestRegister_CreatesNodeTypeAndNode(t *testing.T) {
	b, err := backend.New()
	require.NoError(t, err)

	id, err := sensor.Register(b.GetModel())
	require.NoError(t, err)
	require.NotNil(t, id)

	nt := b.GetModel().GetNodeTypeById(sensor.K8sSensorNodeTypeID)
	require.NotNil(t, nt)
	assert.Equal(t, "k8s-sensor", nt.GetDisplayName())

	n := b.GetModel().GetNodeById(id.NodeID)
	require.NotNil(t, n)
	assert.Equal(t, "k8s-sensor", n.GetDisplayName())
}

func TestRegister_CreatesContextType(t *testing.T) {
	b, err := backend.New()
	require.NoError(t, err)

	_, err = sensor.Register(b.GetModel())
	require.NoError(t, err)

	ct := b.GetModel().GetContextTypeById(sensor.K8sNamespaceContextTypeID)
	require.NotNil(t, ct)
	assert.Equal(t, "Kubernetes Namespace", ct.GetDisplayName())
}

func TestClose_DeletesNode(t *testing.T) {
	b, err := backend.New()
	require.NoError(t, err)

	id, err := sensor.Register(b.GetModel())
	require.NoError(t, err)

	nodeID := id.NodeID
	require.NotNil(t, b.GetModel().GetNodeById(nodeID))

	require.NoError(t, id.Close())
	assert.Nil(t, b.GetModel().GetNodeById(nodeID))
}

func TestClose_Nil(t *testing.T) {
	var id *sensor.Identity
	assert.NoError(t, id.Close())
}
