package controller

import (
	"fmt"

	"github.com/google/uuid"
	"go.emeland.io/modelsrv/pkg/model"
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
// a Finding into the model. CEL runtime errors (e.g. unguarded missing fields) and
// non-bool results are returned as an error, never panic.
func (e *Evaluator) EvaluateRule(rule CompiledRule, obj *unstructured.Unstructured) error {
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

	f, err := buildFinding(rule, obj)
	if err != nil {
		return fmt.Errorf("rule %q: build finding: %w", rule.Name, err)
	}

	if err := e.Model.AddFinding(f); err != nil {
		return fmt.Errorf("rule %q: add finding: %w", rule.Name, err)
	}
	return nil
}

func buildFinding(rule CompiledRule, obj *unstructured.Unstructured) (finding.Finding, error) {
	uid := obj.GetUID()
	if uid == "" {
		return nil, fmt.Errorf("object has no UID")
	}

	id := evaluatorFindingID(rule.Name, uid)
	f := finding.NewFinding(id)
	f.SetDisplayName(rule.Finding.DisplayName)
	f.SetDescription(rule.Finding.Description)

	if typeID := parseOptionalUUID(rule.Finding.Type.UUID); typeID != uuid.Nil {
		f.SetFindingTypeById(typeID)
	}

	applyAnnotations(f.GetAnnotations(), map[string]string{
		annotationSeverity:          rule.Finding.Severity,
		annotationResourceName:      obj.GetName(),
		annotationResourceNamespace: obj.GetNamespace(),
		annotationResourceKind:      obj.GetKind(),
	})

	return f, nil
}
