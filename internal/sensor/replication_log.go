package sensor

import (
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"go.emeland.io/modelsrv/pkg/eventfilter"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
)

// ReplicationPayloadFilter logs every landscape event after other filters run,
// which is the payload the event manager will record and push to subscribers.
func ReplicationPayloadFilter(log logr.Logger) eventfilter.Filter {
	if log.GetSink() == nil {
		log = logr.Discard()
	}
	log = log.WithName("replication-payload")
	return eventfilter.Filter{
		DisplayName: "replication-payload-log",
		Description: "Logs every landscape event that will be recorded and pushed to subscribers",
		Fn: func(_ model.Model, ev events.Event) []events.Event {
			logReplicationEvent(log, ev)
			return []events.Event{ev}
		},
	}
}

func logReplicationEvent(log logr.Logger, ev events.Event) {
	objectTypes := make([]string, 0, len(ev.Objects))
	payloads := make([]string, 0, len(ev.Objects))
	for _, obj := range ev.Objects {
		objectTypes = append(objectTypes, fmt.Sprintf("%T", obj))
		payloads = append(payloads, formatReplicationObject(obj))
	}
	log.Info("replication event leaving filter chain (will be recorded and pushed to subscribers)",
		"resourceType", ev.ResourceType.String(),
		"resourceTypeNum", int(ev.ResourceType),
		"wireKind", ev.ResourceType.WireKind(),
		"operation", ev.Operation.String(),
		"wireOperation", ev.Operation.WireOperation(),
		"resourceId", ev.ResourceId.String(),
		"objectCount", len(ev.Objects),
		"objectTypes", objectTypes,
		"summary", ev.String(),
		"payload", payloads,
	)
	log.V(1).Info("replication event raw dump",
		"resourceId", ev.ResourceId.String(),
		"raw", fmt.Sprintf("%#v", ev),
	)
}

func formatReplicationObject(obj any) string {
	if obj == nil {
		return "<nil>"
	}
	if b, err := json.Marshal(obj); err == nil && string(b) != "{}" && string(b) != "null" {
		return fmt.Sprintf("%T %s", obj, string(b))
	}
	return fmt.Sprintf("%T %#v", obj, obj)
}
