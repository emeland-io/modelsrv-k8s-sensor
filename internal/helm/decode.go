// Package helm decodes Helm v3/v4 release data stored in Kubernetes Secrets.
//
// Helm stores each release revision as a Secret of type helm.sh/release.v1.
// The "release" data field is base64-encoded, gzip-compressed JSON.
// (The K8s client already decodes the outer Secret base64, so we start from
// the raw bytes in Secret.Data["release"].)
package helm

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// magicGzip is the gzip file header.
var magicGzip = []byte{0x1f, 0x8b, 0x08}

// Release holds the decoded fields we care about from a Helm release Secret.
type Release struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Version   int    `json:"version"` // revision number
	Info      Info   `json:"info"`
	Chart     Chart  `json:"chart"`
	Manifest  string `json:"manifest"` // rendered YAML templates
}

// Info holds release status metadata.
type Info struct {
	Status string `json:"status"`
}

// Chart holds chart metadata.
type Chart struct {
	Metadata ChartMetadata `json:"metadata"`
}

// ChartMetadata holds chart name and version.
type ChartMetadata struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// DecodeReleaseData decodes the raw bytes from Secret.Data["release"].
// The data is base64-encoded, then gzip-compressed, then JSON.
func DecodeReleaseData(data []byte) (*Release, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty release data")
	}

	// Helm stores the data as base64 inside the Secret data field.
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	// Decompress if gzipped (for backwards compat, check magic header).
	if len(decoded) >= 3 && bytes.Equal(decoded[:3], magicGzip) {
		r, err := gzip.NewReader(bytes.NewReader(decoded))
		if err != nil {
			return nil, fmt.Errorf("gzip open: %w", err)
		}
		decompressed, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("gzip read: %w", err)
		}
		decoded = decompressed
	}

	var rel Release
	if err := json.Unmarshal(decoded, &rel); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}
	return &rel, nil
}

// ManifestResource represents a single K8s resource found in the rendered manifest.
type ManifestResource struct {
	Kind      string
	Name      string
	Namespace string
}

// ParseManifestResources extracts resource references (kind, name, namespace)
// from the rendered manifest YAML.
func ParseManifestResources(manifest string) []ManifestResource {
	var resources []ManifestResource

	decoder := yaml.NewDecoder(strings.NewReader(manifest))
	for {
		var doc manifestDoc
		if err := decoder.Decode(&doc); err != nil {
			break
		}
		if doc.Kind != "" && doc.Metadata.Name != "" {
			resources = append(resources, ManifestResource{
				Kind:      doc.Kind,
				Name:      doc.Metadata.Name,
				Namespace: doc.Metadata.Namespace,
			})
		}
	}
	return resources
}

// manifestDoc is a minimal struct for YAML unmarshalling of K8s resources.
type manifestDoc struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
}

// HasSystemInstance returns true if the manifest contains a SystemInstance CRD resource.
func HasSystemInstance(resources []ManifestResource) bool {
	for _, r := range resources {
		if r.Kind == "SystemInstance" {
			return true
		}
	}
	return false
}

// SecretNameParts extracts the release name and revision from a Helm Secret name.
// Expected format: sh.helm.release.v1.<release-name>.v<revision>
func SecretNameParts(secretName string) (releaseName string, revision int, ok bool) {
	parts := strings.Split(secretName, ".")
	// sh.helm.release.v1.<name>.v<N> = at least 6 parts
	if len(parts) < 6 {
		return "", 0, false
	}
	if parts[0] != "sh" || parts[1] != "helm" || parts[2] != "release" || parts[3] != "v1" {
		return "", 0, false
	}

	// Release name can contain dots, so rejoin everything between parts[4] and last part.
	last := parts[len(parts)-1]
	if !strings.HasPrefix(last, "v") {
		return "", 0, false
	}
	revStr := strings.TrimPrefix(last, "v")
	rev := 0
	for _, c := range revStr {
		if c < '0' || c > '9' {
			return "", 0, false
		}
		rev = rev*10 + int(c-'0')
	}

	releaseName = strings.Join(parts[4:len(parts)-1], ".")
	return releaseName, rev, true
}
