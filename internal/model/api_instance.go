package model

import (
	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"

	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
)

type APIInstanceInfo struct {
	APIInstance  *modelsrv.APIInstance
	resourceName string
	statusWriter client.SubResourceWriter
}

// AddApiInstance implements Model.
func (m *modelData) AddApiInstance(instance *modelsrv.APIInstance, name string, writer client.SubResourceWriter) error {
	err := m.baseModel.AddApiInstance(instance)
	if err != nil {
		return err
	}

	info := &APIInstanceInfo{
		APIInstance:  instance,
		resourceName: name,
		statusWriter: writer,
	}

	m.ApiInstancesByName[name] = info
	if instance.InstanceId != uuid.Nil {
		m.APIInstancesByUUID[instance.InstanceId] = info
	}
	return nil
}

// GetApiInstanceById implements Model.
func (m *modelData) GetApiInstanceById(id uuid.UUID) *APIInstanceInfo {
	instance, exists := m.APIInstancesByUUID[id]
	if !exists {
		return nil
	}
	return instance
}

// GetApiInstanceByResourceName implements Model.
func (m *modelData) GetApiInstanceByResourceName(s string) *APIInstanceInfo {
	instance, exists := m.ApiInstancesByName[s]
	if !exists {
		return nil
	}
	return instance
}

// DeleteApiInstanceById implements Model.
func (m *modelData) DeleteApiInstanceById(id uuid.UUID) error {
	instanceInfo, exists := m.APIInstancesByUUID[id]
	if !exists {
		return ApiInstanceNotFoundError
	}

	err := m.baseModel.DeleteApiInstanceById(id)

	// remove api instance from by-id map even if error occurs, to clean up
	delete(m.APIInstancesByUUID, id)

	// remove api instance from by-name map even if error occurs, to clean up
	delete(m.ApiInstancesByName, instanceInfo.resourceName)

	return err
}

// DeleteApiInstanceByResourceName implements Model.
func (m *modelData) DeleteApiInstanceByResourceName(s string) error {
	instanceInfo, exists := m.ApiInstancesByName[s]
	if !exists {
		return ApiInstanceNotFoundError
	}

	err := m.baseModel.DeleteApiInstanceById(instanceInfo.APIInstance.InstanceId)

	delete(m.ApiInstancesByName, s)
	if instanceInfo.APIInstance.InstanceId != uuid.Nil {
		delete(m.APIInstancesByUUID, instanceInfo.APIInstance.InstanceId)
	}
	return err
}
