package sensor

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

func TestFormatReplicationObject_Nil(t *testing.T) {
	assert.Equal(t, "<nil>", formatReplicationObject(nil))
}

func TestFormatReplicationObject_JSON(t *testing.T) {
	got := formatReplicationObject(map[string]string{"displayName": "Certificate"})
	assert.Contains(t, got, "Certificate")
}

func TestReplicationPayloadFilter_PassesEventThrough(t *testing.T) {
	id := uuid.New()
	ev := events.Event{
		ResourceType: events.FindingResource,
		Operation:    events.CreateOperation,
		ResourceId:   id,
		Objects:      []any{map[string]string{"displayName": "CRD not available: Certificate"}},
	}
	out := ReplicationPayloadFilter(logr.Discard()).Fn(model.Model(nil), ev)
	require.Len(t, out, 1)
	assert.Equal(t, ev.ResourceId, out[0].ResourceId)
	assert.Equal(t, ev.ResourceType, out[0].ResourceType)
}
