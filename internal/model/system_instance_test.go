package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
)

func TestSystemInstance(t *testing.T) {
	instanceId := uuid.New()
	system := &modelsrv.System{DisplayName: "test-system"}
	sysRef := &modelsrv.SystemRef{System: system}

	inInstance := modelsrv.SystemInstance{
		DisplayName: "test-instance",
		InstanceId:  instanceId,
		SystemRef:   sysRef,
		Annotations: map[string]string{"key": "value"},
	}

	m, err := NewModel()
	assert.NoError(t, err, "NewModel should not return an error")

	err = m.AddSystemInstance(&inInstance, "test-instance", nil)
	assert.NoError(t, err, "AddSystemInstance should not return an error")

	instanceInfo := m.GetSystemInstanceByResourceName("test-instance")
	assert.NotNil(t, instanceInfo, "GetSystemInstanceByResourceName should return a non-nil instance")

	instance := instanceInfo.SystemInstance
	assert.NotNil(t, instance, "SystemInstance should not be nil")

	assert.Equal(t, "test-instance", instance.DisplayName)
	assert.Equal(t, instanceId, instance.InstanceId)
	assert.Equal(t, sysRef, instance.SystemRef)
	assert.Equal(t, "value", instance.Annotations["key"])
}

func TestSystemInstanceOperations(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	instanceId := uuid.New()
	sysRef := modelsrv.SystemRef{System: &modelsrv.System{DisplayName: "test-system"}}
	instance := &modelsrv.SystemInstance{
		DisplayName: "test-instance",
		InstanceId:  instanceId,
		SystemRef:   &sysRef,
	}

	// Test getting non-existent instance
	assert.Nil(t, model.GetSystemInstanceByResourceName("test-instance"))
	assert.Nil(t, model.GetSystemInstanceById(instanceId))

	// Add instance and verify it exists
	err = model.AddSystemInstance(instance, "test-instance", nil)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, instance, model.GetSystemInstanceByResourceName("test-instance").SystemInstance)
	assert.Equal(t, instance, model.GetSystemInstanceById(instanceId).SystemInstance)

	// Delete instance and verify it's gone
	err = model.DeleteSystemInstanceByResourceName("test-instance")
	assert.NoError(t, err)
	assert.Nil(t, model.GetSystemInstanceByResourceName("test-instance"))
	assert.Nil(t, model.GetSystemInstanceById(instanceId))
}

func TestCreateDeleteSystemInstance(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	// Test deleting non-existent instance
	name := "test-instance"
	err = model.DeleteSystemInstanceByResourceName(name)
	assert.Error(t, err)
	assert.Equal(t, SystemInstanceNotFoundError, err)

	instanceId := uuid.New()
	system := &modelsrv.System{DisplayName: "test-system"}
	sysRef := &modelsrv.SystemRef{System: system}

	// Add an invalid SystemInstance
	instance := &modelsrv.SystemInstance{
		DisplayName: name,
		SystemRef:   sysRef,
		Annotations: map[string]string{"key": "value"},
	}
	err = model.AddSystemInstance(instance, name, nil)
	assert.Error(t, err)
	assert.Equal(t, modelsrv.UUIDNotSetError, err)

	// Add a valid SystemInstance
	instance.InstanceId = instanceId

	err = model.AddSystemInstance(instance, name, nil)
	assert.NoError(t, err)
	assert.NotNil(t, model.GetSystemInstanceByResourceName(name))
	assert.NotNil(t, model.GetSystemInstanceById(instanceId))

	// Delete the instance
	err = model.DeleteSystemInstanceByResourceName(name)
	assert.NoError(t, err)

	// Verify instance was deleted
	assert.Nil(t, model.GetSystemInstanceByResourceName(name))
	assert.Nil(t, model.GetSystemInstanceById(instanceId))

	// Try deleting again should return error
	err = model.DeleteSystemInstanceByResourceName(name)
	assert.Error(t, err)
	assert.Equal(t, SystemInstanceNotFoundError, err)
}

func TestDeleteSystemInstanceById(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	// Test deleting non-existent system
	err = model.DeleteSystemInstanceById(uuid.New())
	assert.Equal(t, SystemInstanceNotFoundError, err)

	// Add an invalid system and see it fail to be added
	sys := &modelsrv.SystemInstance{DisplayName: "test-system"}

	err = model.AddSystemInstance(sys, "test-system", nil)
	assert.Error(t, err)
	assert.Equal(t, modelsrv.UUIDNotSetError, err)

	systemId := uuid.New()
	sys.InstanceId = systemId
	err = model.AddSystemInstance(sys, "test-system", nil)
	assert.NoError(t, err)
	assert.NotNil(t, model.GetSystemInstanceByResourceName("test-system"))
	assert.NotNil(t, model.GetSystemInstanceById(systemId))

	// Delete the system
	err = model.DeleteSystemInstanceById(systemId)
	assert.NoError(t, err)

	// Verify system was deleted
	assert.Nil(t, model.GetSystemInstanceByResourceName("test-system"))
	assert.Nil(t, model.GetSystemInstanceById(systemId))

	// Try deleting again should return error
	err = model.DeleteSystemInstanceByResourceName("test-system")
	assert.Equal(t, SystemInstanceNotFoundError, err)
}
