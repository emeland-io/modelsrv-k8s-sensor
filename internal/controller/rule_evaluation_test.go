package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	structurev1alpha1 "gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
	"go.emeland.io/modelsrv/pkg/events"
)

const testNamespaceUID = "11111111-1111-1111-1111-111111111111"

func namespaceFindingRule(name, condition string) *structurev1alpha1.FindingRule {
	return &structurev1alpha1.FindingRule{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: structurev1alpha1.FindingRuleSpec{
			Target: structurev1alpha1.Target{
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
			},
			Condition: condition,
			Finding:   sampleFindingMeta(),
		},
	}
}

func sampleNamespace(name string, uid types.UID) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  uid,
		},
	}
}

func TestRuleEvaluationRunCreatesFinding(t *testing.T) {
	m := testModel(t)
	repo := NewRuleRepo()
	require.NoError(t, repo.Set(namespaceFindingRule("ns-rule", "true")))

	eval := NewEvaluator(m)
	ruleEval := NewRuleEvaluation(repo, eval, "/namespaces")
	ns := sampleNamespace("test-ns", types.UID(testNamespaceUID))

	ruleEval.run(ns)

	findings, err := m.GetFindings()
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Equal(t, "TestFinding", findings[0].GetDisplayName())

	ft, err := findings[0].GetFindingType()
	require.NoError(t, err)
	require.NotNil(t, ft)

	require.Len(t, findings[0].GetResources(), 1)
	assert.Equal(t, events.ContextResource, findings[0].GetResources()[0].ResourceType)
}

func TestRuleEvaluationRunNoMatchingRules(t *testing.T) {
	m := testModel(t)
	repo := NewRuleRepo()
	eval := NewEvaluator(m)
	ruleEval := NewRuleEvaluation(repo, eval, "/namespaces")
	ns := sampleNamespace("test-ns", types.UID(testNamespaceUID))

	ruleEval.run(ns)

	findings, err := m.GetFindings()
	require.NoError(t, err)
	assert.Empty(t, findings)
}

func TestRuleEvaluationRunConditionFalse(t *testing.T) {
	m := testModel(t)
	repo := NewRuleRepo()
	require.NoError(t, repo.Set(namespaceFindingRule("ns-rule", "false")))

	eval := NewEvaluator(m)
	ruleEval := NewRuleEvaluation(repo, eval, "/namespaces")
	ns := sampleNamespace("test-ns", types.UID(testNamespaceUID))

	ruleEval.run(ns)

	findings, err := m.GetFindings()
	require.NoError(t, err)
	assert.Empty(t, findings)
}

func TestRuleEvaluationRunNilReceiver(t *testing.T) {
	var ruleEval *RuleEvaluation
	assert.NotPanics(t, func() {
		ruleEval.run(sampleNamespace("test-ns", types.UID(testNamespaceUID)))
	})
}

func TestRuleEvaluationRunNilFields(t *testing.T) {
	ruleEval := &RuleEvaluation{}
	assert.NotPanics(t, func() {
		ruleEval.run(sampleNamespace("test-ns", types.UID(testNamespaceUID)))
	})
}
