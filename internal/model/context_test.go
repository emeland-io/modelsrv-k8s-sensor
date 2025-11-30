package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
)

func TestContextOperations(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	contextId := uuid.New()
	context := &modelsrv.Context{
		DisplayName: "test-context",
		ContextId:   contextId,
	}

	// Test getting non-existent context
	assert.Nil(t, model.GetContextByResourceName("test-context"))
	assert.Nil(t, model.GetContextById(contextId))

	// Add context and verify it exists
	err = model.AddContext(context, "test-context", nil)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, context, model.GetContextByResourceName("test-context").Context)
	assert.Equal(t, context, model.GetContextById(contextId).Context)

	// Delete context and verify it's gone
	err = model.DeleteContextByResourceName("test-context")
	assert.NoError(t, err)
	assert.Nil(t, model.GetContextByResourceName("test-context"))
	assert.Nil(t, model.GetContextById(contextId))
}

func TestContextDeleteById(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	contextId := uuid.New()
	context := &modelsrv.Context{
		DisplayName: "test-context",
		ContextId:   contextId,
	}

	// Test getting non-existent context
	assert.Nil(t, model.GetContextByResourceName("test-context"))
	assert.Nil(t, model.GetContextById(contextId))

	// Add context and verify it exists
	err = model.AddContext(context, "test-context", nil)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, context, model.GetContextByResourceName("test-context").Context)
	assert.Equal(t, context, model.GetContextById(contextId).Context)

	// Delete context and verify it's gone
	err = model.DeleteContextById(contextId)
	assert.NoError(t, err)
	assert.Nil(t, model.GetContextByResourceName("test-context"))
	assert.Nil(t, model.GetContextById(contextId))
}
