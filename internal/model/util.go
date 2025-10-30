package model

import (
	"time"

	"gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
)

func ParseVersion(v v1alpha1.Version) Version {
	ver := Version{
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
