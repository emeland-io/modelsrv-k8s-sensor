package controller

import (
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/finding"
)

// RegisterRBACFindingTypes registers the well-known FindingType resources for
// RBAC-related findings. These types are always present in the model, even when
// no findings of that type currently exist.
func RegisterRBACFindingTypes(m model.Model) error {
	types := []struct {
		ft          finding.FindingType
		displayName string
		description string
	}{
		{
			ft:          finding.NewFindingType(MissingRoleSpecAnnotationFindingTypeID),
			displayName: "Missing RoleSpec Annotation",
			description: "A Role or ClusterRole does not have the emeland.io/k8s-sensor-role-spec-id annotation set to a valid RoleSpec UUID.",
		},
		{
			ft:          finding.NewFindingType(MissingSubjectAnnotationFindingTypeID),
			displayName: "Missing Subject Annotation",
			description: "A RoleBinding or ClusterRoleBinding does not have the emeland.io/k8s-sensor-subject-id annotation set to a valid Group or Identity UUID.",
		},
	}

	for _, t := range types {
		t.ft.SetDisplayName(t.displayName)
		t.ft.SetDescription(t.description)
		if err := m.AddFindingType(t.ft); err != nil {
			return err
		}
	}
	return nil
}
