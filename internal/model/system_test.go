package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
)

func TestDeleteSystemByResourceName(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	// Test deleting non-existent system
	err = model.DeleteSystemByResourceName("non-existent")
	assert.Equal(t, SystemNotFoundError, err)

	// Add an invalid system and see it fail to be added
	sys := &modelsrv.System{DisplayName: "test-system"}

	err = model.AddSystem(sys, "test-system", nil)
	assert.Error(t, err)
	assert.Equal(t, modelsrv.UUIDNotSetError, err)

	systemId := uuid.New()
	sys.SystemId = systemId
	err = model.AddSystem(sys, "test-system", nil)
	assert.NoError(t, err)
	assert.NotNil(t, model.GetSystemByResourceName("test-system"))
	assert.NotNil(t, model.GetSystemById(systemId))

	// Delete the system
	err = model.DeleteSystemByResourceName("test-system")
	assert.NoError(t, err)

	// Verify system was deleted
	assert.Nil(t, model.GetSystemByResourceName("test-system"))
	assert.Nil(t, model.GetSystemById(systemId))

	// Try deleting again should return error
	err = model.DeleteSystemByResourceName("test-system")
	assert.Equal(t, SystemNotFoundError, err)
}

func TestGetSystemBySystemId(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	sysId := uuid.New()
	sys := &modelsrv.System{
		DisplayName: "test-system",
		SystemId:    sysId,
	}

	// Test getting non-existent system
	assert.Nil(t, model.GetSystemById(sysId))

	// Add system and verify it can be retrieved by UUID
	err = model.AddSystem(sys, "test-system", nil)
	assert.NoError(t, err)

	retrieved := model.GetSystemById(sysId)
	assert.NotNil(t, retrieved)
	assert.Equal(t, sys, retrieved.System)

	retrieved = model.GetSystemByResourceName("test-system")
	assert.NotNil(t, retrieved)
	assert.Equal(t, sys, retrieved.System)
}
