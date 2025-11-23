package controller

import (
	"time"

	"github.com/google/uuid"
	"gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
	modelsrv "gitlab.com/emeland/modelsrv/pkg/model"
)

func parseVersion(v v1alpha1.Version) modelsrv.Version {
	ver := modelsrv.Version{
		Version: v.Version,
	}

	ver.AvailableFrom = parseDate(v.AvailableFrom)
	ver.DeprecatedFrom = parseDate(v.DeprecatedFrom)
	ver.TerminatedFrom = parseDate(v.TerminatedFrom)

	return ver
}

func parseDate(dateStr string) *time.Time {
	if dateStr == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil
	}
	return &t
}

// parseSystemRef parses a SystemRef from either a systemId or a VersionRef
func parseSystemRef(sysId string, sysRef *v1alpha1.VersionRef) *modelsrv.SystemRef {
	if sysId != "" {
		uid, err := uuid.Parse(sysId)
		if err == nil {
			return &modelsrv.SystemRef{
				SystemId: uid,
			}
		}
	}
	if sysRef != nil {
		if sysRef.Name != "" && sysRef.Version != "" {
			return &modelsrv.SystemRef{
				SystemRef: &modelsrv.EntityVersion{
					Name:    sysRef.Name,
					Version: sysRef.Version,
				},
			}
		}
	}

	// no valid reference found
	return nil
}
