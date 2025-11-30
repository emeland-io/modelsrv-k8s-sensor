package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
)

func TestComponentInstanceOperations(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	instanceId := uuid.New()
	componentRef := &modelsrv.ComponentRef{
		Component: &modelsrv.Component{DisplayName: "test-component"},
	}

	instance := &modelsrv.ComponentInstance{
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
	assert.Equal(t, instance, model.GetComponentInstanceByResourceName("test-instance").ComponentInstance)
	assert.Equal(t, instance, model.GetComponentInstanceById(instanceId).ComponentInstance)

	// Delete instance and verify it's gone
	err = model.DeleteComponentInstanceByResourceName("test-instance")
	assert.NoError(t, err)
	assert.Nil(t, model.GetComponentInstanceByResourceName("test-instance"))
	assert.Nil(t, model.GetComponentInstanceById(instanceId))
}
