package sensor_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/emeland/k8s-model/internal/sensor"
)

func TestReplicationGuard_BlocksPushByDefault(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	guard := sensor.ReplicationGuard{Handler: inner}

	req := httptest.NewRequest(http.MethodPost, "/api/events/push", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	guard.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestReplicationGuard_AllowsPushWhenEnabled(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	guard := sensor.ReplicationGuard{Handler: inner, AllowInboundPush: true}

	req := httptest.NewRequest(http.MethodPost, "/api/events/push", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	guard.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestReplicationGuard_PassesOtherRoutes(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	})
	guard := sensor.ReplicationGuard{Handler: inner}

	req := httptest.NewRequest(http.MethodGet, "/api/systems", nil)
	rec := httptest.NewRecorder()
	guard.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}
