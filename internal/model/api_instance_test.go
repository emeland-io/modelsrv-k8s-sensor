package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
)

func TestAPIInstanceOperations(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	instanceId := uuid.New()
	apiRef := &modelsrv.ApiRef{API: &modelsrv.API{DisplayName: "test-api"}}
	instance := &modelsrv.APIInstance{
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
	assert.Equal(t, instance, model.GetApiInstanceByResourceName("test-instance").APIInstance)
	assert.Equal(t, instance, model.GetApiInstanceById(instanceId).APIInstance)

	// Delete instance and verify it's gone
	err = model.DeleteApiInstanceByResourceName("test-instance")
	assert.NoError(t, err)
	assert.Nil(t, model.GetApiInstanceByResourceName("test-instance"))
	assert.Nil(t, model.GetApiInstanceById(instanceId))
}

func TestAPIInstanceDeleteById(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	instanceId := uuid.New()
	apiRef := &modelsrv.ApiRef{API: &modelsrv.API{DisplayName: "test-api"}}
	instance := &modelsrv.APIInstance{
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
	assert.Equal(t, instance, model.GetApiInstanceByResourceName("test-instance").APIInstance)
	assert.Equal(t, instance, model.GetApiInstanceById(instanceId).APIInstance)

	// Delete instance and verify it's gone
	err = model.DeleteApiInstanceById(instanceId)
	assert.NoError(t, err)
	assert.Nil(t, model.GetApiInstanceByResourceName("test-instance"))
	assert.Nil(t, model.GetApiInstanceById(instanceId))
}
