package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/backend"
	"go.emeland.io/modelsrv/pkg/model/finding"
)

func TestRegisterImageFindingTypes(t *testing.T) {
	b, err := backend.New()
	require.NoError(t, err)
	m := b.GetModel()

	require.NoError(t, RegisterImageFindingTypes(m))

	id := finding.TypeIDForKind(ImageNotRetrieved)
	ft := m.GetFindingTypeById(id)
	require.NotNil(t, ft)
	assert.Equal(t, "ImageNotRetrieved", ft.GetDisplayName())

	// Idempotent re-registration.
	require.NoError(t, RegisterImageFindingTypes(m))
}
