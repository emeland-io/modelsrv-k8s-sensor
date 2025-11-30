package model

import (
	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"

	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
)

type ComponentInstanceInfo struct {
	ComponentInstance *modelsrv.ComponentInstance
	resourceName      string
	statusWriter      client.SubResourceWriter
}

// AddComponentInstance implements Model.
func (m *modelData) AddComponentInstance(instance *modelsrv.ComponentInstance, name string, writer client.SubResourceWriter) error {
	err := m.baseModel.AddComponentInstance(instance)
	if err != nil {
		return err
	}

	info := &ComponentInstanceInfo{
		ComponentInstance: instance,
		resourceName:      name,
		statusWriter:      writer,
	}

	m.ComponentInstancesByName[name] = info
	if instance.InstanceId != uuid.Nil {
		m.ComponentInstancesByUUID[instance.InstanceId] = info
	}
	return nil
}

// DeleteComponentInstanceByResourceName implements Model.
func (m *modelData) DeleteComponentInstanceByResourceName(s string) error {
	instanceInfo, exists := m.ComponentInstancesByName[s]
	if !exists {
		return ComponentInstanceNotFoundError
	}

	err := m.baseModel.DeleteComponentInstanceById(instanceInfo.ComponentInstance.InstanceId)

	delete(m.ComponentInstancesByName, s)
	if instanceInfo.ComponentInstance.InstanceId != uuid.Nil {
		delete(m.ComponentInstancesByUUID, instanceInfo.ComponentInstance.InstanceId)
	}
	return err
}

// GetComponentInstanceById implements Model.
func (m *modelData) GetComponentInstanceById(id uuid.UUID) *ComponentInstanceInfo {
	instance, exists := m.ComponentInstancesByUUID[id]
	if !exists {
		return nil
	}
	return instance
}

// GetComponentInstanceByResourceName implements Model.
func (m *modelData) GetComponentInstanceByResourceName(s string) *ComponentInstanceInfo {
	instance, exists := m.ComponentInstancesByName[s]
	if !exists {
		return nil
	}
	return instance
}
