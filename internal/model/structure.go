package model

import (
	"fmt"

	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"

	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
)

var ContextNotFoundError error = fmt.Errorf("Context not found")
var SystemNotFoundError error = fmt.Errorf("System not found")
var ApiNotFoundError error = fmt.Errorf("API not found")
var ComponentNotFoundError error = fmt.Errorf("Component not found")
var SystemInstanceNotFoundError error = fmt.Errorf("System Instance not found")
var ApiInstanceNotFoundError error = fmt.Errorf("API Instance not found")
var ComponentInstanceNotFoundError error = fmt.Errorf("Component Instance not found")

type Model interface {
	AddContext(sys *modelsrv.Context, name string, writer client.SubResourceWriter) error
	DeleteContextByResourceName(s string) error
	DeleteContextById(id uuid.UUID) error
	GetContextById(id uuid.UUID) *ContextInfo

	AddSystem(sys *modelsrv.System, name string, writer client.SubResourceWriter) error
	DeleteSystemByResourceName(s string) error
	DeleteSystemById(id uuid.UUID) error
	GetSystemByResourceName(s string) *SystemInfo
	GetSystemById(id uuid.UUID) *SystemInfo

	AddApi(api *modelsrv.API, name string, writer client.SubResourceWriter) error
	DeleteApiByResourceName(s string) error
	DeleteApiById(id uuid.UUID) error
	GetApiByResourceName(s string) *APIInfo
	GetApiById(id uuid.UUID) *APIInfo

	AddComponent(comp *modelsrv.Component, name string, writer client.SubResourceWriter) error
	DeleteComponentByResourceName(s string) error
	DeleteComponentById(id uuid.UUID) error
	GetComponentByResourceName(s string) *ComponentInfo
	GetComponentById(id uuid.UUID) *ComponentInfo

	AddSystemInstance(instance *modelsrv.SystemInstance, name string, writer client.SubResourceWriter) error
	DeleteSystemInstanceByResourceName(s string) error
	DeleteSystemInstanceById(id uuid.UUID) error
	GetSystemInstanceByResourceName(s string) *SystemInstanceInfo
	GetSystemInstanceById(id uuid.UUID) *SystemInstanceInfo

	AddApiInstance(instance *modelsrv.APIInstance, name string, writer client.SubResourceWriter) error
	DeleteApiInstanceByResourceName(s string) error
	DeleteApiInstanceById(id uuid.UUID) error
	GetApiInstanceByResourceName(s string) *APIInstanceInfo
	GetApiInstanceById(id uuid.UUID) *APIInstanceInfo

	AddComponentInstance(instance *modelsrv.ComponentInstance, name string, writer client.SubResourceWriter) error
	DeleteComponentInstanceByResourceName(s string) error
	GetComponentInstanceByResourceName(s string) *ComponentInstanceInfo
	GetComponentInstanceById(id uuid.UUID) *ComponentInstanceInfo
}

type modelData struct {
	baseModel      modelsrv.Model
	ContextsByUUID map[uuid.UUID]*ContextInfo
	ContextsByName map[string]*ContextInfo

	SystemsByName    map[string]*SystemInfo
	APIsByName       map[string]*APIInfo
	ComponentsByName map[string]*ComponentInfo

	SystemsByUUID    map[uuid.UUID]*SystemInfo
	APIsByUUID       map[uuid.UUID]*APIInfo
	ComponentsByUUID map[uuid.UUID]*ComponentInfo

	SystemInstancesByName    map[string]*SystemInstanceInfo
	ApiInstancesByName       map[string]*APIInstanceInfo
	ComponentInstancesByName map[string]*ComponentInstanceInfo

	SystemInstancesByUUID    map[uuid.UUID]*SystemInstanceInfo
	APIInstancesByUUID       map[uuid.UUID]*APIInstanceInfo
	ComponentInstancesByUUID map[uuid.UUID]*ComponentInstanceInfo
}

// ensure Model interface is implemented correctly
var _ Model = (*modelData)(nil)

func NewModel() (*modelData, error) {
	base, error := modelsrv.NewModel()
	if error != nil {
		return nil, error
	}

	model := &modelData{
		baseModel:      base,
		ContextsByUUID: make(map[uuid.UUID]*ContextInfo),
		ContextsByName: make(map[string]*ContextInfo),

		SystemsByName:    make(map[string]*SystemInfo),
		APIsByName:       make(map[string]*APIInfo),
		ComponentsByName: make(map[string]*ComponentInfo),

		SystemsByUUID:    make(map[uuid.UUID]*SystemInfo),
		APIsByUUID:       make(map[uuid.UUID]*APIInfo),
		ComponentsByUUID: make(map[uuid.UUID]*ComponentInfo),

		SystemInstancesByName:    make(map[string]*SystemInstanceInfo),
		ApiInstancesByName:       make(map[string]*APIInstanceInfo),
		ComponentInstancesByName: make(map[string]*ComponentInstanceInfo),

		SystemInstancesByUUID:    make(map[uuid.UUID]*SystemInstanceInfo),
		APIInstancesByUUID:       make(map[uuid.UUID]*APIInstanceInfo),
		ComponentInstancesByUUID: make(map[uuid.UUID]*ComponentInstanceInfo),
	}

	return model, nil
}

type ContextInfo struct {
	Context      *modelsrv.Context
	resourceName string
	statusWriter client.SubResourceWriter
}

type SystemInfo struct {
	System       *modelsrv.System
	resourceName string
	statusWriter client.SubResourceWriter
}

type APIInfo struct {
	API          *modelsrv.API
	resourceName string
	statusWriter client.SubResourceWriter
}

type ComponentInfo struct {
	Component    *modelsrv.Component
	resourceName string
	statusWriter client.SubResourceWriter
}

type SystemInstanceInfo struct {
	SystemInstance *modelsrv.SystemInstance
	resourceName   string
	statusWriter   client.SubResourceWriter
}

type APIInstanceInfo struct {
	APIInstance  *modelsrv.APIInstance
	resourceName string
	statusWriter client.SubResourceWriter
}

type ComponentInstanceInfo struct {
	ComponentInstance *modelsrv.ComponentInstance
	resourceName      string
	statusWriter      client.SubResourceWriter
}

// AddSystem implements Model.
func (m *modelData) AddSystem(sys *modelsrv.System, name string, statusWriter client.SubResourceWriter) error {

	err := m.baseModel.AddSystem(sys)
	if err != nil {
		return err
	}

	info := &SystemInfo{
		System:       sys,
		resourceName: name,
		statusWriter: statusWriter,
	}

	m.SystemsByName[name] = info
	if sys.SystemId != uuid.Nil {
		m.SystemsByUUID[sys.SystemId] = info
	}

	return nil
}

// DeleteSystemByResourceName implements Model.
func (m *modelData) DeleteSystemByResourceName(s string) error {
	sys, exists := m.SystemsByName[s]
	if !exists {
		return SystemNotFoundError
	}

	// remove system from by-name map even if error occurs, to clean up
	delete(m.SystemsByName, s)

	id := sys.System.SystemId
	if id != uuid.Nil {
		err := m.baseModel.DeleteSystemById(id)

		// remove system-info from by-id map even if error occurs, to clean up
		delete(m.SystemsByUUID, id)
		if err != nil {

			return err
		}
	}

	return nil
}

// GetSystemByResourceName implements Model.
func (m *modelData) GetSystemByResourceName(s string) *SystemInfo {
	system, exists := m.SystemsByName[s]
	if !exists {
		return nil
	}
	return system
}

// GetSystemById implements Model.
func (m *modelData) GetSystemById(id uuid.UUID) *SystemInfo {
	system, exists := m.SystemsByUUID[id]
	if !exists {
		return nil
	}
	return system
}

// AddApi implements Model.
func (m *modelData) AddApi(api *modelsrv.API, name string, writer client.SubResourceWriter) error {
	err := m.baseModel.AddApi(api)
	if err != nil {
		return err
	}

	info := &APIInfo{
		API:          api,
		resourceName: name,
		statusWriter: writer,
	}

	m.APIsByName[name] = info
	if api.ApiId != uuid.Nil {
		m.APIsByUUID[api.ApiId] = info
	}
	return nil
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

// AddComponent implements Model.
func (m *modelData) AddComponent(comp *modelsrv.Component, name string, writer client.SubResourceWriter) error {
	err := m.baseModel.AddComponent(comp)
	if err != nil {
		return err
	}

	info := &ComponentInfo{
		Component:    comp,
		resourceName: name,
		statusWriter: writer,
	}

	m.ComponentsByName[name] = info
	if comp.ComponentId != uuid.Nil {
		m.ComponentsByUUID[comp.ComponentId] = info
	}
	return nil
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

// DeleteApiByResourceName implements Model.
func (m *modelData) DeleteApiByResourceName(s string) error {
	apiInfo, exists := m.APIsByName[s]
	if !exists {
		return ApiNotFoundError
	}

	err := m.baseModel.DeleteApiById(apiInfo.API.ApiId)

	delete(m.APIsByName, s)
	if apiInfo.API.ApiId != uuid.Nil {
		delete(m.APIsByUUID, apiInfo.API.ApiId)
	}
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

// DeleteComponentByResourceName implements Model.
func (m *modelData) DeleteComponentByResourceName(s string) error {
	compInfo, exists := m.ComponentsByName[s]
	if !exists {
		return ComponentNotFoundError
	}

	err := m.baseModel.DeleteComponentById(compInfo.Component.ComponentId)

	delete(m.ComponentsByName, s)
	if compInfo.Component.ComponentId != uuid.Nil {
		delete(m.ComponentsByUUID, compInfo.Component.ComponentId)
	}
	return err
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

// DeleteSystemInstanceByResourceName implements Model.
func (m *modelData) DeleteSystemInstanceByResourceName(s string) error {
	instanceInfo, exists := m.SystemInstancesByName[s]
	if !exists {
		return SystemInstanceNotFoundError
	}

	err := m.baseModel.DeleteSystemInstanceById(instanceInfo.SystemInstance.InstanceId)

	delete(m.SystemInstancesByName, s)
	if instanceInfo.SystemInstance.InstanceId != uuid.Nil {
		delete(m.SystemInstancesByUUID, instanceInfo.SystemInstance.InstanceId)
	}
	return err
}

// GetApiByResourceName implements Model.
func (m *modelData) GetApiByResourceName(s string) *APIInfo {
	api, exists := m.APIsByName[s]
	if !exists {
		return nil
	}
	return api
}

// GetApiById implements Model.
func (m *modelData) GetApiById(id uuid.UUID) *APIInfo {
	api, exists := m.APIsByUUID[id]
	if !exists {
		return nil
	}
	return api
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

// GetComponentById implements Model.
func (m *modelData) GetComponentById(id uuid.UUID) *ComponentInfo {
	comp, exists := m.ComponentsByUUID[id]
	if !exists {
		return nil
	}
	return comp
}

// GetComponentByResourceName implements Model.
func (m *modelData) GetComponentByResourceName(s string) *ComponentInfo {
	comp, exists := m.ComponentsByName[s]
	if !exists {
		return nil
	}
	return comp
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

// AddContext implements Model.
func (m *modelData) AddContext(sys *modelsrv.Context, name string, writer client.SubResourceWriter) error {
	err := m.baseModel.AddContext(sys)
	if err != nil {
		return err
	}

	info := &ContextInfo{
		Context:      sys,
		resourceName: name,
		statusWriter: writer,
	}

	m.ContextsByName[name] = info
	if sys.ContextId != uuid.Nil {
		m.ContextsByUUID[sys.ContextId] = info
	}

	return nil
}

// DeleteApiById implements Model.
func (m *modelData) DeleteApiById(id uuid.UUID) error {
	info, exists := m.APIsByUUID[id]
	if !exists {
		return ApiNotFoundError
	}

	err := m.baseModel.DeleteApiById(id)

	// remove api from by-id map even if error occurs, to clean up
	delete(m.APIsByUUID, id)

	// remove api from by-name map even if error occurs, to clean up
	delete(m.APIsByName, info.resourceName)

	return err
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

// DeleteComponentById implements Model.
func (m *modelData) DeleteComponentById(id uuid.UUID) error {
	compInfo, exists := m.ComponentsByUUID[id]
	if !exists {
		return ComponentNotFoundError
	}

	err := m.baseModel.DeleteComponentById(id)

	// remove component from by-id map even if error occurs, to clean up
	delete(m.ComponentsByUUID, id)

	// remove component from by-name map even if error occurs, to clean up
	delete(m.ComponentsByName, compInfo.resourceName)

	return err
}

// DeleteContextById implements Model.
func (m *modelData) DeleteContextById(id uuid.UUID) error {
	contextInfo, exists := m.ContextsByUUID[id]
	if !exists {
		return ContextNotFoundError
	}

	err := m.baseModel.DeleteContextById(id)

	// remove context from by-id map even if error occurs, to clean up
	delete(m.ContextsByUUID, id)

	// remove context from by-name map even if error occurs, to clean up
	delete(m.ContextsByName, contextInfo.resourceName)

	return err
}

// DeleteContextByResourceName implements Model.
func (m *modelData) DeleteContextByResourceName(s string) error {
	contextInfo, exists := m.ContextsByName[s]
	if !exists {
		return ContextNotFoundError
	}

	// remove context from by-name map even if error occurs, to clean up
	delete(m.ContextsByName, s)

	id := contextInfo.Context.ContextId
	if id != uuid.Nil {
		err := m.baseModel.DeleteContextById(id)

		// remove context-info from by-id map even if error occurs, to clean up
		delete(m.ContextsByUUID, id)
		if err != nil {

			return err
		}
	}

	return nil
}

// DeleteSystemById implements Model.
func (m *modelData) DeleteSystemById(id uuid.UUID) error {
	systemInfo, exists := m.SystemsByUUID[id]
	if !exists {
		return SystemNotFoundError
	}

	err := m.baseModel.DeleteSystemById(id)

	// remove system from by-id map even if error occurs, to clean up
	delete(m.SystemsByUUID, id)

	// remove system from by-name map even if error occurs, to clean up
	delete(m.SystemsByName, systemInfo.resourceName)

	return err
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

// GetContextById implements Model.
func (m *modelData) GetContextById(id uuid.UUID) *ContextInfo {
	context, exists := m.ContextsByUUID[id]
	if !exists {
		return nil
	}
	return context
}
