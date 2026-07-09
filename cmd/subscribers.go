package main

import (
	"fmt"
	"strings"

	"go.emeland.io/modelsrv/pkg/events"
)

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

func registerSubscribers(eventMgr events.EventManager, urls []string) error {
	for _, url := range urls {
		if err := eventMgr.AddSubscriber(url); err != nil {
			return fmt.Errorf("register subscriber %q: %w", url, err)
		}
		setupLog.Info("registered replication subscriber", "url", url)
	}
	return nil
}
