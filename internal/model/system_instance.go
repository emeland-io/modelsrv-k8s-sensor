package model

import (
	"github.com/google/uuid"
	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SystemInstanceInfo struct {
	SystemInstance *modelsrv.SystemInstance
	resourceName   string
	statusWriter   client.SubResourceWriter
}

// AddSystemInstance implements Model.
func (m *modelData) AddSystemInstance(instance *modelsrv.SystemInstance, name string, writer client.SubResourceWriter) error {
	err := m.baseModel.AddSystemInstance(instance)
	if err != nil {
		return err
	}

	info := &SystemInstanceInfo{
		SystemInstance: instance,
		resourceName:   name,
		statusWriter:   writer,
	}

	m.SystemInstancesByName[name] = info
	if instance.InstanceId != uuid.Nil {
		m.SystemInstancesByUUID[instance.InstanceId] = info
	}

	return nil
}

// GetSystemInstanceById implements Model.
func (m *modelData) GetSystemInstanceById(id uuid.UUID) *SystemInstanceInfo {
	instance, exists := m.SystemInstancesByUUID[id]
	if !exists {
		return nil
	}
	return instance
}

// GetSystemInstanceByResourceName implements Model.
func (m *modelData) GetSystemInstanceByResourceName(s string) *SystemInstanceInfo {
	instance, exists := m.SystemInstancesByName[s]
	if !exists {
		return nil
	}
	return instance
}

// DeleteSystemInstanceById implements Model.
func (m *modelData) DeleteSystemInstanceById(id uuid.UUID) error {
	instanceInfo, exists := m.SystemInstancesByUUID[id]
	if !exists {
		return SystemInstanceNotFoundError
	}

	err := m.baseModel.DeleteSystemInstanceById(id)

	// remove system instance from by-id map even if error occurs, to clean up
	delete(m.SystemInstancesByUUID, id)

	// remove system instance from by-name map even if error occurs, to clean up
	delete(m.SystemInstancesByName, instanceInfo.resourceName)

	return err
}
