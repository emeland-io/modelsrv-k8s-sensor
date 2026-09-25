package sensor

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
)

const eventsPushPath = "/api/events/push"
const inboundBodyLimit = 64 * 1024

// ReplicationGuard wraps a modelsrv HTTP handler and rejects inbound event push
// requests so the k8s-sensor acts as a replication source only.
type ReplicationGuard struct {
	Handler          http.Handler
	AllowInboundPush bool
	Log              logr.Logger
}

func (g ReplicationGuard) logger() logr.Logger {
	if g.Log.GetSink() == nil {
		return logr.Discard()
	}
	return g.Log.WithName("replication-inbound")
}

func (g ReplicationGuard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := g.logger()
	isPush := r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, eventsPushPath)
	body := readRequestBody(r, inboundBodyLimit)

	log.V(1).Info("inbound API request",
		"method", r.Method,
		"path", r.URL.Path,
		"rawQuery", r.URL.RawQuery,
		"remote", r.RemoteAddr,
		"host", r.Host,
		"contentType", r.Header.Get("Content-Type"),
		"contentLength", r.ContentLength,
		"userAgent", r.UserAgent(),
		"isPush", isPush,
		"allowInboundPush", g.AllowInboundPush,
	)

	if isPush {
		log.Info("inbound POST /events/push",
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
			"host", r.Host,
			"contentType", r.Header.Get("Content-Type"),
			"contentLength", r.ContentLength,
			"transferEncoding", strings.Join(r.TransferEncoding, ","),
			"allowInboundPush", g.AllowInboundPush,
			"payloadBytes", len(body),
			"payload", string(body),
		)
		if !g.AllowInboundPush {
			log.Info("rejecting inbound event push (sensor is replication source only)",
				"path", r.URL.Path,
				"remote", r.RemoteAddr,
			)
			http.Error(w, "inbound event push is disabled on the k8s sensor", http.StatusForbidden)
			return
		}
		log.Info("forwarding inbound event push to modelsrv handler",
			"path", r.URL.Path,
			"payloadBytes", len(body),
		)
	}

	g.Handler.ServeHTTP(w, r)
}

func readRequestBody(r *http.Request, limit int) []byte {
	if r.Body == nil {
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, int64(limit)+1))
	_ = r.Body.Close()
	if err != nil {
		r.Body = http.NoBody
		return []byte("<read error: " + err.Error() + ">")
	}
	restore := data
	logged := data
	if len(data) > limit {
		restore = data[:limit]
		logged = append(append([]byte(nil), data[:limit]...), []byte("...<truncated>")...)
	}
	r.Body = io.NopCloser(bytes.NewReader(restore))
	return logged
}
