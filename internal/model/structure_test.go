package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
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

func TestAPI(t *testing.T) {
	apiId := uuid.New()
	version := modelsrv.Version{Version: "1.0.0"}
	system := &modelsrv.SystemRef{System: &modelsrv.System{DisplayName: "test-system"}}

	inApi := modelsrv.API{
		DisplayName: "test-api",
		Description: "Test API Description",
		ApiId:       apiId,
		Version:     version,
		Type:        modelsrv.OpenAPI,
		System:      system,
		Annotations: map[string]string{"key": "value"},
	}

	m, err := NewModel()
	assert.NoError(t, err, "NewModel should not return an error")

	err = m.AddApi(&inApi, "test-api", nil)
	assert.NoError(t, err, "AddApi should not return an error")

	outApiInfo := m.GetApiByResourceName("test-api")
	assert.NotNil(t, outApiInfo, "GetApiByResourceName should return a non-nil API")

	outApi := outApiInfo.API

	assert.Equal(t, "test-api", outApi.DisplayName)
	assert.Equal(t, "Test API Description", outApi.Description)
	assert.Equal(t, apiId, outApi.ApiId)
	assert.Equal(t, version, outApi.Version)
	assert.Equal(t, modelsrv.OpenAPI, outApi.Type)
	assert.Equal(t, system, outApi.System)
	assert.Equal(t, "value", outApi.Annotations["key"])
}

func TestComponent(t *testing.T) {
	componentId := uuid.New()
	version := modelsrv.Version{Version: "1.0.0"}
	apiRef := modelsrv.ApiRef{API: &modelsrv.API{DisplayName: "test-api"}}

	inComponent := modelsrv.Component{
		DisplayName: "test-component",
		Description: "Test Component Description",
		ComponentId: componentId,
		Version:     version,
		Consumes:    []modelsrv.ApiRef{apiRef},
		Provides:    []modelsrv.ApiRef{apiRef},
		Annotations: map[string]string{"key": "value"},
	}

	m, err := NewModel()
	assert.NoError(t, err, "NewModel should not return an error")

	err = m.AddComponent(&inComponent, "test-api", nil)
	assert.NoError(t, err, "AddComponent should not return an error")

	outComponentInfo := m.GetComponentByResourceName("test-api")
	assert.NotNil(t, outComponentInfo, "GetApiByResourceName should return a non-nil API")

	outComponent := outComponentInfo.Component

	assert.Equal(t, "test-component", outComponent.DisplayName)
	assert.Equal(t, "Test Component Description", outComponent.Description)
	assert.Equal(t, componentId, outComponent.ComponentId)
	assert.Equal(t, version, outComponent.Version)
	assert.Len(t, outComponent.Consumes, 1)
	assert.Equal(t, apiRef, outComponent.Consumes[0])
	assert.Len(t, outComponent.Provides, 1)
	assert.Equal(t, apiRef, outComponent.Provides[0])
	assert.Equal(t, "value", outComponent.Annotations["key"])
}

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

/*

func TestGetSystemBySystemId(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	sysId := uuid.New()
	sys := &System{
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
	assert.Equal(t, sys, retrieved)
}

func TestAPIOperations(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	apiId := uuid.New()
	api := &API{
		DisplayName: "test-api",
		ApiId:       apiId,
		Type:        OpenAPI,
	}

	// Test getting non-existent API
	assert.Nil(t, model.GetApiByResourceName("test-api"))
	assert.Nil(t, model.GetApiById(apiId))

	// Add API and verify it exists
	err = model.AddApi(api, "test-api", nil)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, api, model.GetApiByResourceName("test-api"))
	assert.Equal(t, api, model.GetApiById(apiId))

	// Delete API and verify it's gone
	err = model.DeleteApiByResourceName("test-api")
	assert.NoError(t, err)
	assert.Nil(t, model.GetApiByResourceName("test-api"))
	assert.Nil(t, model.GetApiById(apiId))
}

func TestComponentOperations(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	componentId := uuid.New()
	component := &Component{
		DisplayName: "test-component",
		ComponentId: componentId,
	}

	// Test getting non-existent component
	assert.Nil(t, model.GetComponentByResourceName("test-component"))
	assert.Nil(t, model.GetComponentById(componentId))

	// Add component and verify it exists
	err = model.AddComponent(component, "test-component", nil)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, component, model.GetComponentByResourceName("test-component"))
	assert.Equal(t, component, model.GetComponentById(componentId))

	// Delete component and verify it's gone
	err = model.DeleteComponentByResourceName("test-component")
	assert.NoError(t, err)
	assert.Nil(t, model.GetComponentByResourceName("test-component"))
	assert.Nil(t, model.GetComponentById(componentId))
}

func TestSystemInstanceOperations(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	instanceId := uuid.New()
	sysRef := SystemRef{System: &System{DisplayName: "test-system"}}
	instance := &SystemInstance{
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
	assert.Equal(t, instance, model.GetSystemInstanceByResourceName("test-instance"))
	assert.Equal(t, instance, model.GetSystemInstanceById(instanceId))

	// Delete instance and verify it's gone
	err = model.DeleteSystemInstanceByResourceName("test-instance")
	assert.NoError(t, err)
	assert.Nil(t, model.GetSystemInstanceByResourceName("test-instance"))
	assert.Nil(t, model.GetSystemInstanceById(instanceId))
}

func TestAPIInstanceOperations(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	instanceId := uuid.New()
	apiRef := ApiRef{API: &API{DisplayName: "test-api"}}
	instance := &APIInstance{
		DisplayName: "test-instance",
		InstanceId:  instanceId,
		ApiRef:      apiRef,
	}

	// Test getting non-existent instance
	assert.Nil(t, model.GetApiInstanceByResourceName("test-instance"))
	assert.Nil(t, model.GetApiInstanceById(instanceId))

	// Add instance and verify it exists
	err = model.AddApiInstance(instance, "test-instance", nil)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, instance, model.GetApiInstanceByResourceName("test-instance"))
	assert.Equal(t, instance, model.GetApiInstanceById(instanceId))

	// Delete instance and verify it's gone
	err = model.DeleteApiInstanceByResourceName("test-instance")
	assert.NoError(t, err)
	assert.Nil(t, model.GetApiInstanceByResourceName("test-instance"))
	assert.Nil(t, model.GetApiInstanceById(instanceId))
}

func TestComponentInstanceOperations(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	instanceId := uuid.New()
	componentRef := EntityVersion{Name: "test-component", Version: "1.0.0"}
	instance := &ComponentInstance{
		DisplayName:  "test-instance",
		InstanceId:   instanceId,
		ComponentRef: componentRef,
	}

	// Test getting non-existent instance
	assert.Nil(t, model.GetComponentInstanceByResourceName("test-instance"))
	assert.Nil(t, model.GetComponentInstanceById(instanceId))

	// Add instance and verify it exists
	err = model.AddComponentInstance(instance, "test-instance", nil)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, instance, model.GetComponentInstanceByResourceName("test-instance"))
	assert.Equal(t, instance, model.GetComponentInstanceById(instanceId))

	// Delete instance and verify it's gone
	err = model.DeleteComponentInstanceByResourceName("test-instance")
	assert.NoError(t, err)
	assert.Nil(t, model.GetComponentInstanceByResourceName("test-instance"))
	assert.Nil(t, model.GetComponentInstanceById(instanceId))
}

func TestApiRef(t *testing.T) {
	apiId := uuid.New()
	api := &API{DisplayName: "test-api"}
	ev := &EntityVersion{Name: "test-api", Version: "1.0.0"}

	apiRef := ApiRef{
		API:    api,
		ApiID:  apiId,
		ApiRef: ev,
	}

	assert.Equal(t, api, apiRef.API)
	assert.Equal(t, apiId, apiRef.ApiID)
	assert.Equal(t, ev, apiRef.ApiRef)
}
*/
