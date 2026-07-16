package controller

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.emeland.io/modelsrv/pkg/events"
	"go.emeland.io/modelsrv/pkg/model"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	structurev1alpha1 "gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
)

func testModel(t *testing.T) model.Model {
	t.Helper()
	m, err := model.NewModel(events.NewDummySink())
	require.NoError(t, err)
	return m
}

func compileTestRule(t *testing.T, condition string, findingMeta structurev1alpha1.Finding) CompiledRule {
	t.Helper()
	program, err := compileCondition(condition)
	require.NoError(t, err)
	return CompiledRule{
		Name:    "test-rule",
		Program: program,
		Finding: findingMeta,
	}
}

func sampleFindingMeta() structurev1alpha1.Finding {
	return structurev1alpha1.Finding{
		Severity:    "high",
		DisplayName: "TestFinding",
		Description: "test description",
		Type: structurev1alpha1.FindingType{
			UUID:        "fa538332-fb6d-51ef-99f3-87831ac140fb",
			DisplayName: "TestType",
		},
	}
}

func sampleUnstructuredObject() *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("v1")
	obj.SetKind("Namespace")
	obj.SetName("test-ns")
	obj.SetUID(types.UID("fa538332-fb6d-51ef-99f3-87831ac140fb"))
	return obj
}

func TestEvaluateRuleConditionTrue(t *testing.T) {
	m := testModel(t)
	eval := NewEvaluator(m)
	rule := compileTestRule(t, "true", sampleFindingMeta())
	obj := sampleUnstructuredObject()

	err := eval.EvaluateRule(rule, obj, events.ContextResource)
	require.NoError(t, err)

	findings, err := m.GetFindings()
	require.NoError(t, err)
	require.Len(t, findings, 1)

	f := findings[0]
	assert.Equal(t, "TestFinding", f.GetDisplayName())
	assert.Equal(t, "test description", f.GetDescription())
	assert.Equal(t, uuid.MustParse("fa538332-fb6d-51ef-99f3-87831ac140fb"), f.GetFindingTypeId())

	ft, err := f.GetFindingType()
	require.NoError(t, err)
	require.NotNil(t, ft)
	assert.Equal(t, "TestType", ft.GetDisplayName())

	require.Len(t, f.GetResources(), 1)
	assert.Equal(t, uuid.MustParse("fa538332-fb6d-51ef-99f3-87831ac140fb"), f.GetResources()[0].ResourceId)
	assert.Equal(t, events.ContextResource, f.GetResources()[0].ResourceType)

	types, err := m.GetFindingTypes()
	require.NoError(t, err)
	require.Len(t, types, 1)
	assert.Equal(t, "TestType", types[0].GetDisplayName())

	assert.Equal(t, evaluatorFindingID("test-rule", obj.GetUID()), f.GetFindingId())

	ann := f.GetAnnotations()
	assert.Equal(t, "high", ann.GetValue(annotationSeverity))
	assert.Equal(t, "test-ns", ann.GetValue(annotationResourceName))
	assert.Equal(t, "", ann.GetValue(annotationResourceNamespace))
	assert.Equal(t, "Namespace", ann.GetValue(annotationResourceKind))
}

func TestEvaluateRuleConditionFalse(t *testing.T) {
	m := testModel(t)
	eval := NewEvaluator(m)
	rule := compileTestRule(t, "false", sampleFindingMeta())
	obj := sampleUnstructuredObject()

	err := eval.EvaluateRule(rule, obj, events.ContextResource)
	require.NoError(t, err)

	findings, err := m.GetFindings()
	require.NoError(t, err)
	assert.Empty(t, findings)
}

func TestEvaluateRuleCELRuntimeError(t *testing.T) {
	m := testModel(t)
	eval := NewEvaluator(m)
	rule := compileTestRule(t, "object.spec.missing.field == 'x'", sampleFindingMeta())
	obj := sampleUnstructuredObject()

	err := eval.EvaluateRule(rule, obj, events.ContextResource)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CEL evaluation failed")

	findings, err := m.GetFindings()
	require.NoError(t, err)
	assert.Empty(t, findings)
}

func TestEvaluateRuleIdempotent(t *testing.T) {
	m := testModel(t)
	eval := NewEvaluator(m)
	rule := compileTestRule(t, "true", sampleFindingMeta())
	obj := sampleUnstructuredObject()

	require.NoError(t, eval.EvaluateRule(rule, obj, events.ContextResource))
	require.NoError(t, eval.EvaluateRule(rule, obj, events.ContextResource))

	findings, err := m.GetFindings()
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Equal(t, evaluatorFindingID("test-rule", obj.GetUID()), findings[0].GetFindingId())
}

func TestEvaluateRuleNilProgram(t *testing.T) {
	m := testModel(t)
	eval := NewEvaluator(m)
	rule := CompiledRule{Name: "test-rule", Finding: sampleFindingMeta()}
	obj := sampleUnstructuredObject()

	err := eval.EvaluateRule(rule, obj, events.ContextResource)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CEL program is nil")
}

func TestEvaluateRuleNilObject(t *testing.T) {
	m := testModel(t)
	eval := NewEvaluator(m)
	rule := compileTestRule(t, "true", sampleFindingMeta())

	err := eval.EvaluateRule(rule, nil, events.ContextResource)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "object is nil")
}

func TestEvaluateRuleNonBoolResult(t *testing.T) {
	m := testModel(t)
	eval := NewEvaluator(m)
	rule := compileTestRule(t, "'not-a-bool'", sampleFindingMeta())
	obj := sampleUnstructuredObject()

	err := eval.EvaluateRule(rule, obj, events.ContextResource)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must return bool")
}

func TestEvaluateRuleInvalidFindingTypeUUID(t *testing.T) {
	m := testModel(t)
	eval := NewEvaluator(m)
	meta := sampleFindingMeta()
	meta.Type.UUID = "not-a-uuid"
	rule := compileTestRule(t, "true", meta)
	obj := sampleUnstructuredObject()

	err := eval.EvaluateRule(rule, obj, events.ContextResource)
	require.NoError(t, err)

	findings, err := m.GetFindings()
	require.NoError(t, err)
	require.Len(t, findings, 1)
	assert.Equal(t, uuid.Nil, findings[0].GetFindingTypeId())
}

func TestEvaluatorFindingIDDeterministic(t *testing.T) {
	id1 := evaluatorFindingID("my-rule", types.UID("uid-1"))
	id2 := evaluatorFindingID("my-rule", types.UID("uid-1"))
	id3 := evaluatorFindingID("other-rule", types.UID("uid-1"))

	assert.Equal(t, id1, id2)
	assert.NotEqual(t, id1, id3)
}
