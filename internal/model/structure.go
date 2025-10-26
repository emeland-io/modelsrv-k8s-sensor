package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var SystemNotFoundError error = fmt.Errorf("System not found")

type Model interface {
	AddSystem(sys *v1alpha1.System, writer client.SubResourceWriter) (any, error)
	DeleteSystemByResourceName(s string) error
}

type modelData struct {
	Systems    []*System
	APIs       []*API
	Components []*Component

	SystemsByUUID    map[uuid.UUID]*System
	APIsByUUID       map[uuid.UUID]*API
	ComponentsByUUID map[uuid.UUID]*Component

	SystemInstances    map[uuid.UUID]*SystemInstance
	APIInstances       map[uuid.UUID]*APIInstance
	ComponentInstances map[uuid.UUID]*ComponentInstance
}

// ensure Model interface is implemented correctly
var _ Model = (*modelData)(nil)

func (m *modelData) AddSystem(sys *v1alpha1.System, writer client.SubResourceWriter) (any, error) {
	panic("unimplemented")
}

func (m *modelData) DeleteSystemByResourceName(s string) error {
	panic("unimplemented")
}

func NewModel() *modelData {
	return &modelData{
		SystemsByUUID:      make(map[uuid.UUID]*System),
		APIsByUUID:         make(map[uuid.UUID]*API),
		ComponentsByUUID:   make(map[uuid.UUID]*Component),
		SystemInstances:    make(map[uuid.UUID]*SystemInstance),
		APIInstances:       make(map[uuid.UUID]*APIInstance),
		ComponentInstances: make(map[uuid.UUID]*ComponentInstance),
	}
}

type Version struct {
	Version        string
	AvailableFrom  time.Time
	DeprecatedFrom time.Time
	TerminatedFrom time.Time
}

type EntityVersion struct {
	Name    string
	Version string
}

type System struct {
	DisplayName string
	Description string
	SystemId    *uuid.UUID
	Version     Version
}

type SystemRef struct {
	System    *System
	SystemID  *uuid.UUID
	SystemRef *EntityVersion
}

type ApiType int

const (
	Unknown ApiType = iota
	Other
	GraphQL
	gRPC
	OpenAPI
)

var ApiTypeValues = map[ApiType]string{
	Unknown: "Unknown",
	OpenAPI: "OpenAPI",
	GraphQL: "GraphQL",
	gRPC:    "gRPC",
	Other:   "Other",
}

func (t ApiType) String() string {
	if val, ok := ApiTypeValues[t]; ok {
		return val
	}
	return ApiTypeValues[Unknown]
}

type API struct {
	DisplayName string
	Description string
	ApiId       *uuid.UUID
	Version     Version
	Type        string
	System      *SystemRef
}

type ApiRef struct {
	API    *API
	ApiID  *uuid.UUID
	ApiRef *EntityVersion
}

type Component struct {
	DisplayName string
	Description string
	ComponentId *uuid.UUID
	Version     Version
	Consumes    []ApiRef
	Provides    []ApiRef
}

type SystemInstance struct {
	DisplayName string
	InstanceId  uuid.UUID
	SystemRef   SystemRef
}

type SystemInstanceRef struct {
	SystemInstance *SystemInstance
	InstanceId     *uuid.UUID
	InstanceRef    *EntityVersion
}

type APIInstance struct {
	DisplayName    string
	InstanceId     uuid.UUID
	ApiRef         ApiRef
	SystemInstance *SystemInstanceRef
}

type ComponentInstance struct {
	DisplayName    string
	InstanceId     uuid.UUID
	ComponentRef   EntityVersion
	SystemInstance *SystemInstanceRef
}
