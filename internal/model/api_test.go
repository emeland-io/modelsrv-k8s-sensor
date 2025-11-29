package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
)

func TestAPI(t *testing.T) {
	apiId := uuid.New()
	version := modelsrv.Version{Version: "1.0.0"}
	system := &modelsrv.SystemRef{System: &modelsrv.System{DisplayName: "test-system"}}

	inApi := modelsrv.API{
		DisplayName: "test-api",
		Description: "Test API Description",
		ApiId:       apiId,
		Version:     version,
		Type:        modelsrv.OpenAPI,
		System:      system,
		Annotations: map[string]string{"key": "value"},
	}

	m, err := NewModel()
	assert.NoError(t, err, "NewModel should not return an error")

	err = m.AddApi(&inApi, "test-api", nil)
	assert.NoError(t, err, "AddApi should not return an error")

	outApiInfo := m.GetApiByResourceName("test-api")
	assert.NotNil(t, outApiInfo, "GetApiByResourceName should return a non-nil API")

	outApi := outApiInfo.API

	assert.Equal(t, "test-api", outApi.DisplayName)
	assert.Equal(t, "Test API Description", outApi.Description)
	assert.Equal(t, apiId, outApi.ApiId)
	assert.Equal(t, version, outApi.Version)
	assert.Equal(t, modelsrv.OpenAPI, outApi.Type)
	assert.Equal(t, system, outApi.System)
	assert.Equal(t, "value", outApi.Annotations["key"])
}

func TestCreateDeleteAPI(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	// Test deleting non-existent API
	name := "test-api"
	err = model.DeleteApiByResourceName(name)
	assert.Error(t, err)
	assert.Equal(t, ApiNotFoundError, err)

	// Add a valid API
	apiId := uuid.New()
	version := modelsrv.Version{Version: "1.0.0"}
	system := &modelsrv.SystemRef{System: &modelsrv.System{DisplayName: "test-system"}}
	api := &modelsrv.API{
		DisplayName: name,
		ApiId:       apiId,
		Version:     version,
		Type:        modelsrv.OpenAPI,
		System:      system,
		Annotations: map[string]string{"key": "value"},
	}

	err = model.AddApi(api, name, nil)
	assert.NoError(t, err)
	assert.NotNil(t, model.GetApiByResourceName(name))
	assert.NotNil(t, model.GetApiById(apiId))

	// Delete the API
	err = model.DeleteApiByResourceName(name)
	assert.NoError(t, err)

	// Verify API was deleted
	assert.Nil(t, model.GetApiByResourceName(name))
	assert.Nil(t, model.GetApiById(apiId))

	// Try deleting again should return error
	err = model.DeleteApiByResourceName(name)
	assert.Error(t, err)
	assert.Equal(t, ApiNotFoundError, err)
}

func TestAPIOperations(t *testing.T) {
	model, err := NewModel()
	assert.NoError(t, err)

	apiId := uuid.New()
	api := &modelsrv.API{
		DisplayName: "test-api",
		ApiId:       apiId,
		Type:        modelsrv.OpenAPI,
	}

	// Test getting non-existent API
	assert.Nil(t, model.GetApiByResourceName("test-api"))
	assert.Nil(t, model.GetApiById(apiId))

	// Add API and verify it exists
	err = model.AddApi(api, "test-api", nil)
	assert.NoError(t, err)

	// Verify retrieval by name and ID
	assert.Equal(t, api, model.GetApiByResourceName("test-api").API)
	assert.Equal(t, api, model.GetApiById(apiId).API)

	// Delete API and verify it's gone
	err = model.DeleteApiByResourceName("test-api")
	assert.NoError(t, err)
	assert.Nil(t, model.GetApiByResourceName("test-api"))
	assert.Nil(t, model.GetApiById(apiId))
}
