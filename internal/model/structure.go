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
	GetContextByResourceName(s string) *ContextInfo

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
