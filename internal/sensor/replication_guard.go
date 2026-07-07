package sensor

import (
	"net/http"
	"strings"
)

const eventsPushPath = "/api/events/push"

// ReplicationGuard wraps a modelsrv HTTP handler and rejects inbound event push
// requests so the k8s-sensor acts as a replication source only.
type ReplicationGuard struct {
	Handler          http.Handler
	AllowInboundPush bool
}

func (g ReplicationGuard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !g.AllowInboundPush && r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, eventsPushPath) {
		http.Error(w, "inbound event push is disabled on the k8s sensor", http.StatusForbidden)
		return
	}
	g.Handler.ServeHTTP(w, r)
}
