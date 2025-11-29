package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
)

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

func TestComponentOperations(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	componentId := uuid.New()
	component := &modelsrv.Component{
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
	assert.Equal(t, component, model.GetComponentByResourceName("test-component").Component)
	assert.Equal(t, component, model.GetComponentById(componentId).Component)

	// Delete component and verify it's gone
	err = model.DeleteComponentByResourceName("test-component")
	assert.NoError(t, err)
	assert.Nil(t, model.GetComponentByResourceName("test-component"))
	assert.Nil(t, model.GetComponentById(componentId))
}
