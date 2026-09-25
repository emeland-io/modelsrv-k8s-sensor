package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"go.emeland.io/modelsrv/pkg/events"
)

const subscriberProbeTimeout = 3 * time.Second
const subscriberProbeBodyLimit = 2048

// parseCommaSeparatedList splits a comma-separated string into trimmed non-empty entries.
func parseCommaSeparatedList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if s := strings.TrimSpace(part); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func registerSubscribers(eventMgr events.EventManager, rawFlag string) error {
	log := setupLog.WithName("subscribers")
	envRaw, envSet := os.LookupEnv("SUBSCRIBER_URLS")
	urls := parseCommaSeparatedList(rawFlag)

	log.Info("subscriber configuration",
		"flagOrResolved", rawFlag,
		"envSUBSCRIBER_URLS_set", envSet,
		"envSUBSCRIBER_URLS", envRaw,
		"parsedCount", len(urls),
		"parsedURLs", urls,
	)

	if len(urls) == 0 {
		log.Info("no replication subscribers configured; landscape events will stay local")
		return nil
	}

	for i, raw := range urls {
		inspectSubscriberURL(log, i, raw)
		probeSubscriber(log, raw)

		before := snapshotSubscribers(eventMgr)
		seqBefore, seqBeforeErr := eventMgr.GetCurrentSequenceId(context.Background())
		log.Info("AddSubscriber starting",
			"index", i,
			"url", raw,
			"alreadyRegistered", before,
			"sequenceBefore", seqBefore,
			"sequenceBeforeErr", errString(seqBeforeErr),
		)

		if err := eventMgr.AddSubscriber(raw); err != nil {
			log.Error(err, "AddSubscriber failed",
				"index", i,
				"url", raw,
				"errType", fmt.Sprintf("%T", err),
			)
			return fmt.Errorf("register subscriber %q: %w", raw, err)
		}

		after := snapshotSubscribers(eventMgr)
		seqAfter, seqAfterErr := eventMgr.GetCurrentSequenceId(context.Background())
		log.Info("AddSubscriber succeeded (modelsrv also replays current state to this URL)",
			"index", i,
			"url", raw,
			"registered", after,
			"sequenceAfter", seqAfter,
			"sequenceAfterErr", errString(seqAfterErr),
		)
	}

	log.Info("all replication subscribers registered",
		"count", len(eventMgr.GetSubscribers()),
		"registered", snapshotSubscribers(eventMgr),
	)
	return nil
}

func inspectSubscriberURL(log logr.Logger, index int, raw string) {
	log.Info("inspecting subscriber URL", "index", index, "raw", raw, "rawBytes", []byte(raw))
	u, err := url.Parse(raw)
	if err != nil {
		log.Error(err, "subscriber URL failed to parse", "index", index, "raw", raw)
		return
	}
	host, port, splitErr := net.SplitHostPort(u.Host)
	if splitErr != nil {
		host = u.Host
		port = ""
		if u.Scheme == "http" {
			port = "80 (implied)"
		}
		if u.Scheme == "https" {
			port = "443 (implied)"
		}
	}
	log.Info("subscriber URL parsed",
		"index", index,
		"raw", raw,
		"scheme", u.Scheme,
		"opaque", u.Opaque,
		"userSet", u.User != nil,
		"host", host,
		"port", port,
		"path", u.Path,
		"rawPath", u.RawPath,
		"query", u.RawQuery,
		"fragment", u.Fragment,
		"requestURI", u.RequestURI(),
	)
	switch {
	case u.Scheme != "http" && u.Scheme != "https":
		log.Info("subscriber URL scheme is not http/https; Linkerd/proxy may not treat this as mesh traffic",
			"index", index, "scheme", u.Scheme, "raw", raw)
	case u.Host == "":
		log.Info("subscriber URL has empty host", "index", index, "raw", raw)
	}
	if u.Path != "" && !strings.Contains(u.Path, "/api") {
		log.Info("subscriber URL path does not contain /api (modelsrv expects a base API URL like http://host:8080/api)",
			"index", index, "path", u.Path, "raw", raw)
	}
}

func probeSubscriber(log logr.Logger, raw string) {
	client := &http.Client{Timeout: subscriberProbeTimeout}
	ctx, cancel := context.WithTimeout(context.Background(), subscriberProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		log.Error(err, "subscriber probe: could not build GET request", "url", raw)
		return
	}
	log.Info("subscriber probe: GET",
		"url", raw,
		"method", req.Method,
		"host", req.URL.Host,
		"path", req.URL.Path,
		"timeout", subscriberProbeTimeout.String(),
	)
	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		log.Error(err, "subscriber probe: GET failed (proxy/mesh/DNS/TLS often shows up here)",
			"url", raw,
			"elapsed", elapsed.String(),
			"errType", fmt.Sprintf("%T", err),
		)
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Error(closeErr, "subscriber probe: closing response body", "url", raw)
		}
	}()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, subscriberProbeBodyLimit))
	// GET to a modelsrv /api base is often 404 even when replication is
	// healthy. This probe never blocks registration (3s timeout, log only).
	log.Info("subscriber probe: GET response",
		"url", raw,
		"elapsed", elapsed.String(),
		"status", resp.Status,
		"statusCode", resp.StatusCode,
		"proto", resp.Proto,
		"contentType", resp.Header.Get("Content-Type"),
		"server", resp.Header.Get("Server"),
		"via", resp.Header.Get("Via"),
		"l5dClientId", resp.Header.Get("l5d-client-id"),
		"payloadBytes", len(body),
	)
	log.V(1).Info("subscriber probe: GET response body",
		"url", raw,
		"statusCode", resp.StatusCode,
		"payloadBytes", len(body),
		"payload", string(body),
	)
}

func snapshotSubscribers(eventMgr events.EventManager) []map[string]string {
	subs := eventMgr.GetSubscribers()
	out := make([]map[string]string, 0, len(subs))
	for _, s := range subs {
		if s == nil {
			out = append(out, map[string]string{"status": "nil"})
			continue
		}
		out = append(out, map[string]string{
			"id":     s.GetId().String(),
			"url":    s.GetURL(),
			"status": s.GetStatus(),
		})
	}
	return out
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
