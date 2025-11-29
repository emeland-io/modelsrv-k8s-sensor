package model

import (
	"github.com/google/uuid"
	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type APIInfo struct {
	API          *modelsrv.API
	resourceName string
	statusWriter client.SubResourceWriter
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
