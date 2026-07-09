package controller

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	structurev1alpha1 "gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
)

func sampleFindingRule(name, condition string) *structurev1alpha1.FindingRule {
	return &structurev1alpha1.FindingRule{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: structurev1alpha1.FindingRuleSpec{
			Target: structurev1alpha1.Target{
				APIGroups: []string{"", "apps"},
				Resources: []string{"namespaces", "deployments"},
			},
			Condition: condition,
			Finding: structurev1alpha1.Finding{
				Severity:    "high",
				DisplayName: "TestFinding",
				Description: "test description",
				Type: structurev1alpha1.FindingType{
					UUID:        "uuid-1",
					DisplayName: "TestType",
				},
			},
		},
	}
}

func TestRuleRepoSetAndGet(t *testing.T) {
	repo := NewRuleRepo()
	rule := sampleFindingRule("test-rule", "true")

	err := repo.Set(rule)
	require.NoError(t, err)

	keys := []string{"/namespaces", "/deployments", "apps/namespaces", "apps/deployments"}
	for _, key := range keys {
		rules := repo.GetRulesForResource(key)
		require.Len(t, rules, 1)
		assert.Equal(t, "test-rule", rules[0].Name)
		assert.Equal(t, "high", rules[0].Finding.Severity)
		assert.NotNil(t, rules[0].Program)
	}
}

func TestRuleRepoInvalidCEL(t *testing.T) {
	repo := NewRuleRepo()
	rule := sampleFindingRule("bad-rule", "this is not valid cel !!!")

	err := repo.Set(rule)
	require.Error(t, err)
	assert.Empty(t, repo.GetRulesForResource("/namespaces"))
}

func TestRuleRepoUpdateInPlace(t *testing.T) {
	repo := NewRuleRepo()
	rule := sampleFindingRule("test-rule", "true")
	require.NoError(t, repo.Set(rule))

	updated := sampleFindingRule("test-rule", "false")
	updated.Spec.Finding.Severity = "low"
	require.NoError(t, repo.Set(updated))

	rules := repo.GetRulesForResource("/namespaces")
	require.Len(t, rules, 1)
	assert.Equal(t, "low", rules[0].Finding.Severity)
}

func TestRuleRepoRemove(t *testing.T) {
	repo := NewRuleRepo()
	rule := sampleFindingRule("test-rule", "true")
	require.NoError(t, repo.Set(rule))

	repo.Remove("test-rule")

	assert.Empty(t, repo.GetRulesForResource("/namespaces"))
	assert.Empty(t, repo.GetRulesForResource("apps/deployments"))
}

func TestRuleRepoGetUnknownResource(t *testing.T) {
	repo := NewRuleRepo()
	assert.Nil(t, repo.GetRulesForResource("unknown/resource"))
}

func TestRuleRepoConcurrent(t *testing.T) {
	repo := NewRuleRepo()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := "rule-" + string(rune('a'+i%26))
			rule := sampleFindingRule(name, "true")
			_ = repo.Set(rule)
			_ = repo.GetRulesForResource("/namespaces")
			repo.Remove(name)
		}(i)
	}
	wg.Wait()
}
