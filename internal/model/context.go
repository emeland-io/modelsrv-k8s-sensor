package model

import (
	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"

	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
)

type ContextInfo struct {
	Context      *modelsrv.Context
	resourceName string
	statusWriter client.SubResourceWriter
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

// GetContextById implements Model.
func (m *modelData) GetContextById(id uuid.UUID) *ContextInfo {
	context, exists := m.ContextsByUUID[id]
	if !exists {
		return nil
	}
	return context
}

// GetContextByResourceName implements Model.
func (m *modelData) GetContextByResourceName(s string) *ContextInfo {
	context, exists := m.ContextsByName[s]
	if !exists {
		return nil
	}
	return context
}
