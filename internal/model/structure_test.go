package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestVersion(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)

	version := Version{
		Version:        "1.0.0",
		AvailableFrom:  &now,
		DeprecatedFrom: &future,
		TerminatedFrom: nil,
	}

	assert.Equal(t, "1.0.0", version.Version)
	assert.Equal(t, now, *version.AvailableFrom)
	assert.Equal(t, future, *version.DeprecatedFrom)
	assert.Nil(t, version.TerminatedFrom)
}

func TestEntityVersion(t *testing.T) {
	ev := EntityVersion{
		Name:    "test-entity",
		Version: "1.0.0",
	}

	assert.Equal(t, "test-entity", ev.Name)
	assert.Equal(t, "1.0.0", ev.Version)
}

func TestSystem(t *testing.T) {
	sysId := uuid.New()
	version := Version{Version: "1.0.0"}

	system := System{
		DisplayName: "test-system",
		Description: "Test System Description",
		SystemId:    sysId,
		Version:     version,
		Abstract:    false,
		Annotations: map[string]string{"key": "value"},
	}

	assert.Equal(t, "test-system", system.DisplayName)
	assert.Equal(t, "Test System Description", system.Description)
	assert.Equal(t, sysId, system.SystemId)
	assert.Equal(t, version, system.Version)
	assert.False(t, system.Abstract)
	assert.Equal(t, "value", system.Annotations["key"])
}

func TestSystemRef(t *testing.T) {
	sysId := uuid.New()
	system := &System{DisplayName: "test-system"}
	ev := &EntityVersion{Name: "test-system", Version: "1.0.0"}

	sysRef := SystemRef{
		System:    system,
		SystemId:  sysId,
		SystemRef: ev,
	}

	assert.Equal(t, system, sysRef.System)
	assert.Equal(t, sysId, sysRef.SystemId)
	assert.Equal(t, ev, sysRef.SystemRef)
}

func TestApiType(t *testing.T) {
	tests := []struct {
		apiType  ApiType
		expected string
	}{
		{Unknown, "Unknown"},
		{OpenAPI, "OpenAPI"},
		{GraphQL, "GraphQL"},
		{gRPC, "gRPC"},
		{Other, "Other"},
		{ApiType(99), "Unknown"}, // Invalid value should return Unknown
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.apiType.String(),
			"ApiType.String() should return correct string representation")
	}
}

func TestAPI(t *testing.T) {
	apiId := uuid.New()
	version := Version{Version: "1.0.0"}
	system := &SystemRef{System: &System{DisplayName: "test-system"}}

	api := API{
		DisplayName: "test-api",
		Description: "Test API Description",
		ApiId:       apiId,
		Version:     version,
		Type:        OpenAPI,
		System:      system,
		Annotations: map[string]string{"key": "value"},
	}

	assert.Equal(t, "test-api", api.DisplayName)
	assert.Equal(t, "Test API Description", api.Description)
	assert.Equal(t, apiId, api.ApiId)
	assert.Equal(t, version, api.Version)
	assert.Equal(t, OpenAPI, api.Type)
	assert.Equal(t, system, api.System)
	assert.Equal(t, "value", api.Annotations["key"])
}

func TestComponent(t *testing.T) {
	componentId := uuid.New()
	version := Version{Version: "1.0.0"}
	apiRef := ApiRef{API: &API{DisplayName: "test-api"}}

	component := Component{
		DisplayName: "test-component",
		Description: "Test Component Description",
		ComponentId: componentId,
		Version:     version,
		Consumes:    []ApiRef{apiRef},
		Provides:    []ApiRef{apiRef},
		Annotations: map[string]string{"key": "value"},
	}

	assert.Equal(t, "test-component", component.DisplayName)
	assert.Equal(t, "Test Component Description", component.Description)
	assert.Equal(t, componentId, component.ComponentId)
	assert.Equal(t, version, component.Version)
	assert.Len(t, component.Consumes, 1)
	assert.Equal(t, apiRef, component.Consumes[0])
	assert.Len(t, component.Provides, 1)
	assert.Equal(t, apiRef, component.Provides[0])
	assert.Equal(t, "value", component.Annotations["key"])
}

func TestSystemInstance(t *testing.T) {
	instanceId := uuid.New()
	system := &System{DisplayName: "test-system"}
	sysRef := &SystemRef{System: system}

	instance := SystemInstance{
		DisplayName: "test-instance",
		InstanceId:  instanceId,
		SystemRef:   sysRef,
		Annotations: map[string]string{"key": "value"},
	}

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
	assert.ErrorIs(t, err, ErrSystemNotFound)

	// Add a system and verify it exists
	sys := &System{DisplayName: "test-system"}
	err = model.AddSystem(sys, "test-system", nil)
	assert.NoError(t, err)
	assert.NotNil(t, model.GetSystemByResourceName("test-system"))

	// Delete the system
	err = model.DeleteSystemByResourceName("test-system")
	assert.NoError(t, err)

	// Verify system was deleted
	assert.Nil(t, model.GetSystemByResourceName("test-system"))

	// Try deleting again should return error
	err = model.DeleteSystemByResourceName("test-system")
	assert.ErrorIs(t, err, ErrSystemNotFound)
}

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

func TestAddOverwritesByName(t *testing.T) {
	m, err := NewModel()
	assert.NoError(t, err)

	id1 := uuid.New()
	id2 := uuid.New()

	// System: add then overwrite with same name
	assert.NoError(t, m.AddSystem(&System{DisplayName: "v1", SystemId: id1}, "sys", nil))
	assert.NoError(t, m.AddSystem(&System{DisplayName: "v2", SystemId: id2}, "sys", nil))
	assert.Equal(t, "v2", m.GetSystemByResourceName("sys").DisplayName)
	assert.NotNil(t, m.GetSystemById(id2))
	// Old UUID still points to stale entry (no cleanup on overwrite)
	assert.NotNil(t, m.GetSystemById(id1))

	// API
	assert.NoError(t, m.AddApi(&API{DisplayName: "v1", ApiId: id1}, "api", nil))
	assert.NoError(t, m.AddApi(&API{DisplayName: "v2", ApiId: id2}, "api", nil))
	assert.Equal(t, "v2", m.GetApiByResourceName("api").DisplayName)

	// Component
	assert.NoError(t, m.AddComponent(&Component{DisplayName: "v1", ComponentId: id1}, "comp", nil))
	assert.NoError(t, m.AddComponent(&Component{DisplayName: "v2", ComponentId: id2}, "comp", nil))
	assert.Equal(t, "v2", m.GetComponentByResourceName("comp").DisplayName)
}

func TestDeleteNonExistentReturnsError(t *testing.T) {
	m, err := NewModel()
	assert.NoError(t, err)

	assert.ErrorIs(t, m.DeleteSystemByResourceName("x"), ErrSystemNotFound)
	assert.ErrorIs(t, m.DeleteApiByResourceName("x"), ErrApiNotFound)
	assert.ErrorIs(t, m.DeleteComponentByResourceName("x"), ErrComponentNotFound)
	assert.ErrorIs(t, m.DeleteSystemInstanceByResourceName("x"), ErrSystemInstanceNotFound)
	assert.ErrorIs(t, m.DeleteApiInstanceByResourceName("x"), ErrApiInstanceNotFound)
	assert.ErrorIs(t, m.DeleteComponentInstanceByResourceName("x"), ErrComponentInstanceNotFound)
	assert.ErrorIs(t, m.DeleteContextByResourceName("x"), ErrContextNotFound)
}

func TestAddWithNilUUID(t *testing.T) {
	m, err := NewModel()
	assert.NoError(t, err)

	// Entities with uuid.Nil should be stored by name but not by UUID.
	assert.NoError(t, m.AddSystem(&System{DisplayName: "s", SystemId: uuid.Nil}, "s", nil))
	assert.NotNil(t, m.GetSystemByResourceName("s"))
	assert.Nil(t, m.GetSystemById(uuid.Nil))

	assert.NoError(t, m.AddApi(&API{DisplayName: "a", ApiId: uuid.Nil}, "a", nil))
	assert.NotNil(t, m.GetApiByResourceName("a"))
	assert.Nil(t, m.GetApiById(uuid.Nil))

	assert.NoError(t, m.AddComponent(&Component{DisplayName: "c", ComponentId: uuid.Nil}, "c", nil))
	assert.NotNil(t, m.GetComponentByResourceName("c"))
	assert.Nil(t, m.GetComponentById(uuid.Nil))

	assert.NoError(t, m.AddSystemInstance(&SystemInstance{DisplayName: "si", InstanceId: uuid.Nil}, "si", nil))
	assert.NotNil(t, m.GetSystemInstanceByResourceName("si"))
	assert.Nil(t, m.GetSystemInstanceById(uuid.Nil))

	assert.NoError(t, m.AddApiInstance(&APIInstance{DisplayName: "ai", InstanceId: uuid.Nil}, "ai", nil))
	assert.NotNil(t, m.GetApiInstanceByResourceName("ai"))
	assert.Nil(t, m.GetApiInstanceById(uuid.Nil))

	assert.NoError(t, m.AddComponentInstance(&ComponentInstance{DisplayName: "ci", InstanceId: uuid.Nil}, "ci", nil))
	assert.NotNil(t, m.GetComponentInstanceByResourceName("ci"))
	assert.Nil(t, m.GetComponentInstanceById(uuid.Nil))
}

func TestDeleteCleansUpUUIDMap(t *testing.T) {
	m, err := NewModel()
	assert.NoError(t, err)

	id := uuid.New()

	assert.NoError(t, m.AddSystem(&System{SystemId: id}, "s", nil))
	assert.NoError(t, m.DeleteSystemByResourceName("s"))
	assert.Nil(t, m.GetSystemById(id))

	assert.NoError(t, m.AddApi(&API{ApiId: id}, "a", nil))
	assert.NoError(t, m.DeleteApiByResourceName("a"))
	assert.Nil(t, m.GetApiById(id))

	assert.NoError(t, m.AddComponent(&Component{ComponentId: id}, "c", nil))
	assert.NoError(t, m.DeleteComponentByResourceName("c"))
	assert.Nil(t, m.GetComponentById(id))

	assert.NoError(t, m.AddSystemInstance(&SystemInstance{InstanceId: id}, "si", nil))
	assert.NoError(t, m.DeleteSystemInstanceByResourceName("si"))
	assert.Nil(t, m.GetSystemInstanceById(id))

	assert.NoError(t, m.AddApiInstance(&APIInstance{InstanceId: id}, "ai", nil))
	assert.NoError(t, m.DeleteApiInstanceByResourceName("ai"))
	assert.Nil(t, m.GetApiInstanceById(id))

	assert.NoError(t, m.AddComponentInstance(&ComponentInstance{InstanceId: id}, "ci", nil))
	assert.NoError(t, m.DeleteComponentInstanceByResourceName("ci"))
	assert.Nil(t, m.GetComponentInstanceById(id))
}

func TestDoubleDeleteReturnsError(t *testing.T) {
	m, err := NewModel()
	assert.NoError(t, err)

	id := uuid.New()

	assert.NoError(t, m.AddSystem(&System{SystemId: id}, "s", nil))
	assert.NoError(t, m.DeleteSystemByResourceName("s"))
	assert.ErrorIs(t, m.DeleteSystemByResourceName("s"), ErrSystemNotFound)

	assert.NoError(t, m.AddApi(&API{ApiId: id}, "a", nil))
	assert.NoError(t, m.DeleteApiByResourceName("a"))
	assert.ErrorIs(t, m.DeleteApiByResourceName("a"), ErrApiNotFound)

	assert.NoError(t, m.AddContext(&Context{ContextId: id}, "ctx"))
	assert.NoError(t, m.DeleteContextByResourceName("ctx"))
	assert.ErrorIs(t, m.DeleteContextByResourceName("ctx"), ErrContextNotFound)
}

func TestVersionIsEqual(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)

	v1 := Version{Version: "1.0", AvailableFrom: &now}
	v2 := Version{Version: "1.0", AvailableFrom: &now}
	v3 := Version{Version: "1.0", AvailableFrom: &later}
	v4 := Version{Version: "2.0", AvailableFrom: &now}
	v5 := Version{Version: "1.0", AvailableFrom: nil}

	assert.True(t, v1.IsEqual(v2))
	assert.False(t, v1.IsEqual(v3)) // different date
	assert.False(t, v1.IsEqual(v4)) // different version string
	assert.False(t, v1.IsEqual(v5)) // nil vs non-nil
	assert.False(t, v5.IsEqual(v1)) // non-nil vs nil
}

func TestParseApiType(t *testing.T) {
	assert.Equal(t, OpenAPI, ParseApiType("OpenAPI"))
	assert.Equal(t, GraphQL, ParseApiType("GraphQL"))
	assert.Equal(t, Unknown, ParseApiType("bogus"))
	assert.Equal(t, Unknown, ParseApiType(""))
}

func TestAddContextNilAnnotations(t *testing.T) {
	m, err := NewModel()
	assert.NoError(t, err)

	// Context with nil annotations should not panic.
	assert.NoError(t, m.AddContext(&Context{
		DisplayName: "bare",
		ContextId:   uuid.New(),
		Annotations: nil,
	}, "bare"))
	assert.NotNil(t, m.GetContextByResourceName("bare"))
}
