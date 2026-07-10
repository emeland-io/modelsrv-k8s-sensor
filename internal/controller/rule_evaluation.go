package controller

import (
	"go.emeland.io/modelsrv/pkg/events"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	logr "sigs.k8s.io/controller-runtime/pkg/log"
)

func modelsrvResourceType(registryKey string) events.ResourceType {
	switch registryKey {
	case "/namespaces":
		return events.ContextResource
	case "/services", "networking.k8s.io/ingresses":
		return events.APIInstanceResource
	case "apps/deployments", "apps/statefulsets", "apps/daemonsets", "batch/cronjobs", "batch/jobs":
		return events.ComponentInstanceResource
	case "structure.emeland.io/systems":
		return events.SystemResource
	case "structure.emeland.io/apis":
		return events.APIResource
	case "structure.emeland.io/components":
		return events.ComponentResource
	case "structure.emeland.io/systeminstances":
		return events.SystemInstanceResource
	default:
		return events.UnknownResourceType
	}
}

// RuleEvaluation binds a rule registry, CEL evaluator, and resource type key
// for evaluating FindingRules against reconciled objects.
type RuleEvaluation struct {
	Rules        *RuleRepo
	Evaluator    *Evaluator
	ResourceType string
}

// NewRuleEvaluation creates a RuleEvaluation for the given group/resource key.
func NewRuleEvaluation(repo *RuleRepo, eval *Evaluator, resourceType string) *RuleEvaluation {
	return &RuleEvaluation{
		Rules:        repo,
		Evaluator:    eval,
		ResourceType: resourceType,
	}
}

// run looks up rules for this resource type, converts the typed object to
// unstructured, and evaluates each matching rule.
func (e *RuleEvaluation) run(obj runtime.Object) {
	if e == nil || e.Rules == nil || e.Evaluator == nil {
		return
	}

	rules := e.Rules.GetRulesForResource(e.ResourceType)
	if len(rules) == 0 {
		return
	}

	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		logr.Log.WithName("rule-evaluation").Error(
			err,
			"failed to convert object to unstructured",
			"resourceType", e.ResourceType,
		)
		return
	}

	u := &unstructured.Unstructured{Object: m}
	subjectResourceType := modelsrvResourceType(e.ResourceType)
	log := logr.Log.WithName("rule-evaluation")
	for _, rule := range rules {
		if err := e.Evaluator.EvaluateRule(rule, u, subjectResourceType); err != nil {
			log.Error(err, "rule evaluation failed", "rule", rule.Name, "resourceType", e.ResourceType)
		}
	}
}
