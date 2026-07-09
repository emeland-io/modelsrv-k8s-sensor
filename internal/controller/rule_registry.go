package controller

import (
	"fmt"
	"sync"

	"github.com/google/cel-go/cel"
	structurev1alpha1 "gitlab.com/emeland/k8s-model/api/k8s/v1alpha1"
)

// CompiledRule holds a compiled CEL program and the finding metadata from a FindingRule.
type CompiledRule struct {
	Name    string
	Program cel.Program
	Finding structurev1alpha1.Finding
}

// RuleRepo is a thread-safe in-memory registry of compiled FindingRules,
// keyed by target resource type (group/resource).
type RuleRepo struct {
	mu    sync.RWMutex
	rules map[string][]CompiledRule
}

var ruleEnv = mustNewRuleEnv()

func mustNewRuleEnv() *cel.Env {
	env, err := cel.NewEnv(cel.Variable("object", cel.DynType))
	if err != nil {
		panic(fmt.Sprintf("failed to create CEL environment: %v", err))
	}
	return env
}

// compileCondition compiles a CEL expression against the shared rule environment.
func compileCondition(condition string) (cel.Program, error) {
	ast, iss := ruleEnv.Compile(condition)
	if iss.Err() != nil {
		return nil, iss.Err()
	}
	program, err := ruleEnv.Program(ast)
	if err != nil {
		return nil, err
	}
	return program, nil
}

// NewRuleRepo creates an empty rule registry.
func NewRuleRepo() *RuleRepo {
	return &RuleRepo{
		rules: make(map[string][]CompiledRule),
	}
}

// resourceKey returns the registry key for a group/resource pair.
func resourceKey(group, resource string) string {
	return group + "/" + resource
}

// targetKeys returns all group/resource keys for a FindingRule target.
func targetKeys(target structurev1alpha1.Target) []string {
	keys := make([]string, 0, len(target.APIGroups)*len(target.Resources))
	for _, group := range target.APIGroups {
		for _, resource := range target.Resources {
			keys = append(keys, resourceKey(group, resource))
		}
	}
	return keys
}

// Set compiles the rule's condition and stores it under each target group/resource key.
// Any existing entries for the same rule name are replaced.
func (r *RuleRepo) Set(rule *structurev1alpha1.FindingRule) error {
	program, err := compileCondition(rule.Spec.Condition)
	if err != nil {
		return err
	}

	compiled := CompiledRule{
		Name:    rule.Name,
		Program: program,
		Finding: rule.Spec.Finding,
	}

	keys := targetKeys(rule.Spec.Target)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.removeLocked(rule.Name)
	for _, key := range keys {
		r.rules[key] = append(r.rules[key], compiled)
	}
	return nil
}

// Remove deletes all registry entries for the given rule name.
func (r *RuleRepo) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.removeLocked(name)
}

func (r *RuleRepo) removeLocked(name string) {
	for key, rules := range r.rules {
		filtered := rules[:0]
		for _, rule := range rules {
			if rule.Name != name {
				filtered = append(filtered, rule)
			}
		}
		if len(filtered) == 0 {
			delete(r.rules, key)
		} else {
			r.rules[key] = filtered
		}
	}
}

// GetRulesForResource returns a copy of compiled rules for the given resource type key.
func (r *RuleRepo) GetRulesForResource(resourceType string) []CompiledRule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rules, ok := r.rules[resourceType]
	if !ok {
		return nil
	}
	out := make([]CompiledRule, len(rules))
	copy(out, rules)
	return out
}
