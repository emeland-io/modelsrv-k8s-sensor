package helm

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// encodeTestRelease creates a realistic Secret.Data["release"] payload for testing.
func encodeTestRelease(t *testing.T, rel *Release) []byte {
	t.Helper()
	data, err := EncodeRelease(rel)
	require.NoError(t, err)
	return data
}

func TestDecodeReleaseData(t *testing.T) {
	rel := &Release{
		Name:      "my-app",
		Namespace: "production",
		Version:   3,
		Info:      Info{Status: "deployed"},
		Chart: Chart{
			Metadata: ChartMetadata{Name: "my-chart", Version: "1.2.3"},
		},
		Manifest: "---\napiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: my-app\n  namespace: production\n",
	}
	data := encodeTestRelease(t, rel)

	got, err := DecodeReleaseData(data)
	require.NoError(t, err)
	assert.Equal(t, "my-app", got.Name)
	assert.Equal(t, "production", got.Namespace)
	assert.Equal(t, 3, got.Version)
	assert.Equal(t, "deployed", got.Info.Status)
	assert.Equal(t, "my-chart", got.Chart.Metadata.Name)
	assert.Equal(t, "1.2.3", got.Chart.Metadata.Version)
	assert.Contains(t, got.Manifest, "Deployment")
}

func TestDecodeReleaseData_Empty(t *testing.T) {
	_, err := DecodeReleaseData(nil)
	assert.Error(t, err)

	_, err = DecodeReleaseData([]byte{})
	assert.Error(t, err)
}

func TestDecodeReleaseData_InvalidBase64(t *testing.T) {
	_, err := DecodeReleaseData([]byte("not-valid-base64!!!"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "base64")
}

func TestDecodeReleaseData_InvalidJSON(t *testing.T) {
	// Valid base64+gzip but not JSON
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write([]byte("not json"))
	_ = gz.Close()
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	_, err := DecodeReleaseData([]byte(encoded))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "json")
}

func TestParseManifestResources(t *testing.T) {
	manifest := `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
  namespace: production
---
apiVersion: v1
kind: Service
metadata:
  name: frontend-svc
  namespace: production
---
apiVersion: batch/v1
kind: Job
metadata:
  name: migration
  namespace: production
`
	resources := ParseManifestResources(manifest)
	require.Len(t, resources, 3)

	assert.Equal(t, "Deployment", resources[0].Kind)
	assert.Equal(t, "frontend", resources[0].Name)
	assert.Equal(t, "production", resources[0].Namespace)

	assert.Equal(t, "Service", resources[1].Kind)
	assert.Equal(t, "frontend-svc", resources[1].Name)

	assert.Equal(t, "Job", resources[2].Kind)
	assert.Equal(t, "migration", resources[2].Name)
}

func TestParseManifestResources_Empty(t *testing.T) {
	assert.Empty(t, ParseManifestResources(""))
	assert.Empty(t, ParseManifestResources("---\n---\n"))
}

func TestParseManifestResources_NameInLabels(t *testing.T) {
	// "name" appears in labels but the actual metadata.name should win.
	manifest := `---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    name: label-value
    app: my-app
  name: actual-name
  namespace: ns
`
	resources := ParseManifestResources(manifest)
	require.Len(t, resources, 1)
	assert.Equal(t, "actual-name", resources[0].Name)
	assert.Equal(t, "ns", resources[0].Namespace)
}

func TestHasSystemInstance(t *testing.T) {
	resources := []ManifestResource{ //nolint:prealloc
		{Kind: "Deployment", Name: "app"},
		{Kind: "Service", Name: "app-svc"},
	}
	assert.False(t, HasSystemInstance(resources))

	resources = append(resources, ManifestResource{Kind: "SystemInstance", Name: "my-si"})
	assert.True(t, HasSystemInstance(resources))
}

func TestSecretNameParts(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantName string
		wantRev  int
		wantOK   bool
	}{
		{"simple", "sh.helm.release.v1.my-app.v3", "my-app", 3, true},
		{"dotted name", "sh.helm.release.v1.my.dotted.app.v12", "my.dotted.app", 12, true},
		{"revision 1", "sh.helm.release.v1.nginx.v1", "nginx", 1, true},
		{"too short", "sh.helm.release.v1.v1", "", 0, false},
		{"wrong prefix", "io.helm.release.v1.app.v1", "", 0, false},
		{"no revision prefix", "sh.helm.release.v1.app.3", "", 0, false},
		{"non-numeric revision", "sh.helm.release.v1.app.vfoo", "", 0, false},
		{"empty", "", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, rev, ok := SecretNameParts(tt.input)
			assert.Equal(t, tt.wantOK, ok)
			if ok {
				assert.Equal(t, tt.wantName, name)
				assert.Equal(t, tt.wantRev, rev)
			}
		})
	}
}
