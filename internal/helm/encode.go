package helm

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// EncodeRelease encodes a Release struct into the format stored in
// Secret.Data["release"] (JSON -> gzip -> base64). This is the inverse of
// DecodeReleaseData and is useful for testing.
func EncodeRelease(rel *Release) ([]byte, error) {
	jsonBytes, err := json.Marshal(rel)
	if err != nil {
		return nil, fmt.Errorf("json marshal: %w", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(jsonBytes); err != nil {
		return nil, fmt.Errorf("gzip write: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return []byte(encoded), nil
}
