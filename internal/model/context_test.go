package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestContextOperations(t *testing.T) {
	m, err := NewModel()
	assert.NoError(t, err)

	ctxId := uuid.New()
	parentId := uuid.New()
	ctx := &Context{
		DisplayName: "test-namespace",
		ContextId:   ctxId,
		Description: "Kubernetes namespace test-namespace",
		ParentId:    parentId,
		Annotations: map[string]string{"key": "value"},
	}

	// Get non-existent
	assert.Nil(t, m.GetContextByResourceName("test-namespace"))
	assert.Nil(t, m.GetContextById(ctxId))

	// Add and verify
	err = m.AddContext(ctx, "test-namespace")
	assert.NoError(t, err)
	assert.Equal(t, ctx, m.GetContextByResourceName("test-namespace"))
	assert.Equal(t, ctx, m.GetContextById(ctxId))
	assert.Equal(t, parentId, m.GetContextById(ctxId).ParentId)
	assert.Equal(t, "value", m.GetContextById(ctxId).Annotations["key"])

	// Update (re-add with same name)
	updated := &Context{
		DisplayName: "updated-namespace",
		ContextId:   ctxId,
	}
	err = m.AddContext(updated, "test-namespace")
	assert.NoError(t, err)
	assert.Equal(t, "updated-namespace", m.GetContextByResourceName("test-namespace").DisplayName)

	// Delete
	err = m.DeleteContextByResourceName("test-namespace")
	assert.NoError(t, err)
	assert.Nil(t, m.GetContextByResourceName("test-namespace"))
	assert.Nil(t, m.GetContextById(ctxId))

	// Delete non-existent
	err = m.DeleteContextByResourceName("test-namespace")
	assert.Equal(t, ContextNotFoundError, err)
}

func TestContextWithoutUUID(t *testing.T) {
	m, err := NewModel()
	assert.NoError(t, err)

	ctx := &Context{
		DisplayName: "no-uuid",
		ContextId:   uuid.Nil,
	}

	err = m.AddContext(ctx, "no-uuid")
	assert.NoError(t, err)
	assert.Equal(t, ctx, m.GetContextByResourceName("no-uuid"))
	// Should not be retrievable by UUID
	assert.Nil(t, m.GetContextById(uuid.Nil))
}

func TestNewModelInitializesContextMaps(t *testing.T) {
	m, err := NewModel()
	assert.NoError(t, err)
	assert.NotNil(t, m.ContextsByName)
	assert.NotNil(t, m.ContextsByUUID)
}
