package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewModel(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err, "NewModel should not return an error")
	assert.NotNil(t, model, "NewModel should return a non-nil model")

	// Verify all maps are initialized
	assert.NotNil(t, model.SystemsByName, "SystemsByName map should be initialized")
	assert.NotNil(t, model.APIsByName, "APIsByName map should be initialized")
	assert.NotNil(t, model.ComponentsByName, "ComponentsByName map should be initialized")
	assert.NotNil(t, model.SystemsByUUID, "SystemsByUUID map should be initialized")
	assert.NotNil(t, model.APIsByUUID, "APIsByUUID map should be initialized")
	assert.NotNil(t, model.ComponentsByUUID, "ComponentsByUUID map should be initialized")
	assert.NotNil(t, model.SystemInstancesByUUID, "SystemInstances map should be initialized")
	assert.NotNil(t, model.APIInstancesByUUID, "APIInstances map should be initialized")
	assert.NotNil(t, model.ComponentInstancesByUUID, "ComponentInstances map should be initialized")
}
