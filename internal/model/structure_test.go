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
	assert.NotNil(t, model.SystemInstances, "SystemInstances map should be initialized")
	assert.NotNil(t, model.APIInstances, "APIInstances map should be initialized")
	assert.NotNil(t, model.ComponentInstances, "ComponentInstances map should be initialized")
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
		SystemId:    &sysId,
		Version:     version,
		Abstract:    false,
		Annotations: map[string]string{"key": "value"},
	}

	assert.Equal(t, "test-system", system.DisplayName)
	assert.Equal(t, "Test System Description", system.Description)
	assert.Equal(t, sysId, *system.SystemId)
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
		SystemID:  &sysId,
		SystemRef: ev,
	}

	assert.Equal(t, system, sysRef.System)
	assert.Equal(t, sysId, *sysRef.SystemID)
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
		ApiId:       &apiId,
		Version:     version,
		Type:        OpenAPI.String(),
		System:      system,
		Annotations: map[string]string{"key": "value"},
	}

	assert.Equal(t, "test-api", api.DisplayName)
	assert.Equal(t, "Test API Description", api.Description)
	assert.Equal(t, apiId, *api.ApiId)
	assert.Equal(t, version, api.Version)
	assert.Equal(t, OpenAPI.String(), api.Type)
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
		ComponentId: &componentId,
		Version:     version,
		Consumes:    []ApiRef{apiRef},
		Provides:    []ApiRef{apiRef},
		Annotations: map[string]string{"key": "value"},
	}

	assert.Equal(t, "test-component", component.DisplayName)
	assert.Equal(t, "Test Component Description", component.Description)
	assert.Equal(t, componentId, *component.ComponentId)
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
	sysRef := SystemRef{System: system}

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
	assert.Equal(t, SystemNotFoundError, err)

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
	assert.Equal(t, SystemNotFoundError, err)
}
