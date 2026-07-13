package controller

import (
	"fmt"

	"github.com/google/uuid"
	structurev1alpha1 "gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"go.emeland.io/modelsrv/pkg/model/common"
	"go.emeland.io/modelsrv/pkg/model/finding"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

const (
	annotationSeverity          = "severity"
	annotationResourceName      = "resourceName"
	annotationResourceNamespace = "resourceNamespace"
	annotationResourceKind      = "resourceKind"
)

// evaluatorFindingNamespace is the UUID v5 namespace for deterministic finding IDs.
var evaluatorFindingNamespace = uuid.MustParse("bdd8f2ac-5b06-47d4-981b-19850ec2c344")

// Evaluator runs compiled CEL rules against Kubernetes resources and raises
// findings in the modelsrv model when conditions match.
type Evaluator struct {
	Model model.Model
}

// NewEvaluator creates an evaluator backed by the given model.
func NewEvaluator(m model.Model) *Evaluator {
	return &Evaluator{Model: m}
}

func evaluatorFindingID(ruleName string, subjectUID types.UID) uuid.UUID {
	key := append([]byte(ruleName+"/"), []byte(subjectUID)...)
	return uuid.NewSHA1(evaluatorFindingNamespace, key)
}

// EvaluateRule evaluates rule.Program against obj (bound as CEL variable "object",
// matching the env in rule_registry.go). If the result is true, builds and pushes
// a Finding into the model. subjectResourceType identifies the modelsrv resource
// kind for linking the subject object in Finding.Resources.
// CEL runtime errors (e.g. unguarded missing fields) and non-bool results are
// returned as an error, never panic.
func (e *Evaluator) EvaluateRule(rule CompiledRule, obj *unstructured.Unstructured, subjectResourceType events.ResourceType) error {
	if rule.Program == nil {
		return fmt.Errorf("rule %q: CEL program is nil", rule.Name)
	}
	if obj == nil {
		return fmt.Errorf("rule %q: object is nil", rule.Name)
	}

	out, _, err := rule.Program.Eval(map[string]interface{}{
		"object": obj.Object,
	})
	if err != nil {
		return fmt.Errorf("rule %q: CEL evaluation failed: %w", rule.Name, err)
	}

	val := out.Value()
	matched, ok := val.(bool)
	if !ok {
		return fmt.Errorf("rule %q: CEL expression must return bool, got %T", rule.Name, val)
	}
	if !matched {
		return nil
	}

	f, err := buildFinding(e.Model, rule, obj, subjectResourceType)
	if err != nil {
		return fmt.Errorf("rule %q: build finding: %w", rule.Name, err)
	}

	if err := e.Model.AddFinding(f); err != nil {
		return fmt.Errorf("rule %q: add finding: %w", rule.Name, err)
	}
	return nil
}

func ensureFindingType(m model.Model, meta structurev1alpha1.FindingType) (finding.FindingType, error) {
	typeID := parseOptionalUUID(meta.UUID)
	if typeID == uuid.Nil {
		return nil, nil
	}

	if ft := m.GetFindingTypeById(typeID); ft != nil {
		return ft, nil
	}

	ft := finding.NewFindingType(typeID)
	ft.SetDisplayName(meta.DisplayName)
	if meta.Description != "" {
		ft.SetDescription(meta.Description)
	}
	if err := m.AddFindingType(ft); err != nil {
		if existing := m.GetFindingTypeById(typeID); existing != nil {
			return existing, nil
		}
		return nil, err
	}
	return ft, nil
}

func buildFinding(m model.Model, rule CompiledRule, obj *unstructured.Unstructured, subjectResourceType events.ResourceType) (finding.Finding, error) {
	uid := obj.GetUID()
	if uid == "" {
		return nil, fmt.Errorf("object has no UID")
	}

	id := evaluatorFindingID(rule.Name, uid)
	f := finding.NewFinding(id)
	f.SetDisplayName(rule.Finding.DisplayName)
	f.SetDescription(rule.Finding.Description)

	ft, err := ensureFindingType(m, rule.Finding.Type)
	if err != nil {
		return nil, fmt.Errorf("ensure finding type: %w", err)
	}
	if ft != nil {
		f.SetFindingTypeByRef(ft)
	}

	subjectID := parseOptionalUUID(string(uid))
	if subjectID != uuid.Nil && subjectResourceType != events.UnknownResourceType {
		f.SetResources([]*common.ResourceRef{{
			ResourceId:   subjectID,
			ResourceType: subjectResourceType,
		}})
	}

	applyAnnotations(f.GetAnnotations(), map[string]string{
		annotationSeverity:          rule.Finding.Severity,
		annotationResourceName:      obj.GetName(),
		annotationResourceNamespace: obj.GetNamespace(),
		annotationResourceKind:      obj.GetKind(),
	})

	return f, nil
}
