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

/*


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
