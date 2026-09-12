package filter_test

import (
	"testing"

	"github.com/gusandrioli/tfp/internal/filter"
	"github.com/gusandrioli/tfp/internal/planmodel"
)

func diffAt(pathStr ...string) planmodel.AttributeDiff {
	path := make(planmodel.AttributePath, len(pathStr))
	for i, k := range pathStr {
		path[i] = planmodel.PathSegment{Key: k}
	}
	return planmodel.AttributeDiff{Path: path}
}

func TestSet_GlobalRuleHidesAcrossResourceTypes(t *testing.T) {
	var s filter.Set
	s.Add(filter.Rule{
		Scope:      filter.ScopeGlobal,
		PathSuffix: planmodel.AttributePath{{Key: "labels"}, {Key: "app.kubernetes.io/version"}},
	})

	d := diffAt("metadata", "labels", "app.kubernetes.io/version")
	if !s.Hides("kubernetes_role_binding", d) {
		t.Error("expected the global rule to hide this diff on kubernetes_role_binding")
	}
	if !s.Hides("kubernetes_deployment", d) {
		t.Error("expected the global rule to also hide it on a different resource type")
	}

	other := diffAt("metadata", "labels", "team")
	if s.Hides("kubernetes_role_binding", other) {
		t.Error("did not expect the rule to hide an unrelated label key")
	}
}

func TestSet_ResourceTypeScopedRule(t *testing.T) {
	var s filter.Set
	s.Add(filter.Rule{
		Scope:        filter.ScopeResourceType,
		ResourceType: "kubernetes_role_binding",
		PathSuffix:   planmodel.AttributePath{{Key: "labels"}, {Key: "app.kubernetes.io/version"}},
	})

	d := diffAt("metadata", "labels", "app.kubernetes.io/version")
	if !s.Hides("kubernetes_role_binding", d) {
		t.Error("expected the scoped rule to hide this diff on its resource type")
	}
	if s.Hides("kubernetes_deployment", d) {
		t.Error("did not expect the scoped rule to hide it on a different resource type")
	}
}

func TestSet_RemoveAndClear(t *testing.T) {
	var s filter.Set
	s.Add(filter.Rule{Scope: filter.ScopeGlobal, PathSuffix: planmodel.AttributePath{{Key: "a"}}})
	s.Add(filter.Rule{Scope: filter.ScopeGlobal, PathSuffix: planmodel.AttributePath{{Key: "b"}}})

	s.Remove(0)
	rules := s.Rules()
	if len(rules) != 1 || rules[0].PathSuffix.String() != "b" {
		t.Fatalf("after Remove(0), rules = %+v, want just the \"b\" rule", rules)
	}

	s.Clear()
	if len(s.Rules()) != 0 {
		t.Fatal("expected Clear to remove all rules")
	}
}

func TestSet_NilSetHidesNothing(t *testing.T) {
	var s *filter.Set
	if s.Hides("any_type", diffAt("a", "b")) {
		t.Error("a nil *Set must hide nothing")
	}
}

func TestSet_RulesReturnsACopy(t *testing.T) {
	var s filter.Set
	s.Add(filter.Rule{Scope: filter.ScopeGlobal, PathSuffix: planmodel.AttributePath{{Key: "a"}}})

	rules := s.Rules()
	rules[0].Scope = filter.ScopeResourceType

	if got := s.Rules()[0].Scope; got != filter.ScopeGlobal {
		t.Fatalf("mutating the returned slice affected the Set: Scope = %v, want ScopeGlobal", got)
	}
}
