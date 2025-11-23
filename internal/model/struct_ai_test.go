package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
)

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
		InstanceId:  instanceId,
		SystemRef:   sysRef,
		Annotations: map[string]string{"key": "value"},
	}
	err = model.AddSystemInstance(instance, name, nil)
	assert.Error(t, err)

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

func TestCreateDeleteAPI(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	// Test deleting non-existent API
	name := "test-api"
	err = model.DeleteApiByResourceName(name)
	assert.Error(t, err)
	assert.Equal(t, ApiNotFoundError, err)

	// Add a valid API
	apiId := uuid.New()
	version := modelsrv.Version{Version: "1.0.0"}
	system := &modelsrv.SystemRef{System: &modelsrv.System{DisplayName: "test-system"}}
	api := &modelsrv.API{
		DisplayName: name,
		ApiId:       apiId,
		Version:     version,
		Type:        modelsrv.OpenAPI,
		System:      system,
		Annotations: map[string]string{"key": "value"},
	}

	err = model.AddApi(api, name, nil)
	assert.NoError(t, err)
	assert.NotNil(t, model.GetApiByResourceName(name))
	assert.NotNil(t, model.GetApiById(apiId))

	// Delete the API
	err = model.DeleteApiByResourceName(name)
	assert.NoError(t, err)

	// Verify API was deleted
	assert.Nil(t, model.GetApiByResourceName(name))
	assert.Nil(t, model.GetApiById(apiId))

	// Try deleting again should return error
	err = model.DeleteApiByResourceName(name)
	assert.Error(t, err)
	assert.Equal(t, ApiNotFoundError, err)
}

func TestCreateDeleteComponent(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	// Test deleting non-existent Component
	name := "test-component"
	err = model.DeleteComponentByResourceName(name)
	assert.Error(t, err)
	assert.Equal(t, ComponentNotFoundError, err)

	// Add a valid Component
	componentId := uuid.New()
	version := modelsrv.Version{Version: "1.0.0"}
	component := &modelsrv.Component{
		DisplayName: name,
		ComponentId: componentId,
		Version:     version,
		Annotations: map[string]string{"key": "value"},
	}

	err = model.AddComponent(component, name, nil)
	assert.NoError(t, err)
	assert.NotNil(t, model.GetComponentByResourceName(name))
	assert.NotNil(t, model.GetComponentById(componentId))

	// Delete the Component
	err = model.DeleteComponentByResourceName(name)
	assert.NoError(t, err)

	// Verify Component was deleted
	assert.Nil(t, model.GetComponentByResourceName(name))
	assert.Nil(t, model.GetComponentById(componentId))

	// Try deleting again should return error
	err = model.DeleteComponentByResourceName(name)
	assert.Error(t, err)
	assert.Equal(t, ComponentNotFoundError, err)
}
