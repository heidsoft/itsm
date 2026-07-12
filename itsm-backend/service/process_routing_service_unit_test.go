package service

import "testing"

func TestProcessRoutingConditionsOperatorsAndNestedValues(t *testing.T) {
	svc := &ProcessRoutingService{}
	variables := map[string]interface{}{
		"risk":        map[string]interface{}{"score": 85.0},
		"environment": "production",
		"amount":      1200,
	}
	tests := []struct {
		name       string
		conditions map[string]interface{}
		want       bool
	}{
		{"nested gte", map[string]interface{}{"risk.score": map[string]interface{}{"gte": 80}}, true},
		{"in", map[string]interface{}{"environment": map[string]interface{}{"in": []interface{}{"production", "staging"}}}, true},
		{"not in", map[string]interface{}{"environment": map[string]interface{}{"notIn": []interface{}{"testing"}}}, true},
		{"numeric range", map[string]interface{}{"amount": map[string]interface{}{"gt": 1000, "lte": 2000}}, true},
		{"missing allowed", map[string]interface{}{"optional": map[string]interface{}{"exists": false}}, true},
		{"unknown operator fails closed", map[string]interface{}{"amount": map[string]interface{}{"script": "true"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := svc.evaluateConditions(tt.conditions, variables); got != tt.want {
				t.Fatalf("evaluateConditions()=%v want %v", got, tt.want)
			}
		})
	}
}

// TestEvaluateConditions_RejectsUnknownOperator asserts that every operator
// that the routing engine does not recognize fails the whole condition. The
// behaviour is fail-closed: the binding does not match unless an explicit
// implementation for the operator is added.
func TestEvaluateConditions_RejectsUnknownOperator(t *testing.T) {
	svc := &ProcessRoutingService{}
	variables := map[string]interface{}{"amount": 50}
	cases := []string{"script", "regex", "contains", "matches", "between", "weird"}
	for _, op := range cases {
		t.Run(op, func(t *testing.T) {
			conds := map[string]interface{}{
				"amount": map[string]interface{}{op: "anything"},
			}
			if got := svc.evaluateConditions(conds, variables); got != false {
				t.Fatalf("operator %q must fail-closed, got %v", op, got)
			}
		})
	}
}

// TestEvaluateConditions_NestedMissingPath covers the case where the dot-path
// starts on a non-object value (e.g. a scalar) or traverses through one. The
// lookup must return false rather than panic.
func TestEvaluateConditions_NestedMissingPath(t *testing.T) {
	svc := &ProcessRoutingService{}

	t.Run("scalar at root", func(t *testing.T) {
		variables := map[string]interface{}{"a": "scalar"}
		conds := map[string]interface{}{"a.b.c": map[string]interface{}{"eq": "x"}}
		if got := svc.evaluateConditions(conds, variables); got != false {
			t.Fatalf("scalar path traversal must return false, got %v", got)
		}
	})
	t.Run("nil at intermediate", func(t *testing.T) {
		variables := map[string]interface{}{"a": map[string]interface{}{"b": nil}}
		conds := map[string]interface{}{"a.b.c": map[string]interface{}{"eq": "x"}}
		if got := svc.evaluateConditions(conds, variables); got != false {
			t.Fatalf("nil intermediate must return false, got %v", got)
		}
	})
	t.Run("missing intermediate key", func(t *testing.T) {
		variables := map[string]interface{}{"a": map[string]interface{}{}}
		conds := map[string]interface{}{"a.b.c": map[string]interface{}{"eq": "x"}}
		if got := svc.evaluateConditions(conds, variables); got != false {
			t.Fatalf("missing intermediate key must return false, got %v", got)
		}
	})
	t.Run("exists=false on missing intermediate is allowed", func(t *testing.T) {
		variables := map[string]interface{}{"a": map[string]interface{}{}}
		conds := map[string]interface{}{"a.b.c": map[string]interface{}{"exists": false}}
		if got := svc.evaluateConditions(conds, variables); got != true {
			t.Fatalf("exists=false on missing path must be true, got %v", got)
		}
	})
}

// TestEvaluateConditions_NumericCoercion verifies that comparisons between
// int, float64 and numeric strings evaluate consistently.
func TestEvaluateConditions_NumericCoercion(t *testing.T) {
	svc := &ProcessRoutingService{}

	cases := []struct {
		name       string
		variables  map[string]interface{}
		conditions map[string]interface{}
		want       bool
	}{
		{
			name:       "int vs float64 gte",
			variables:  map[string]interface{}{"v": 100},
			conditions: map[string]interface{}{"v": map[string]interface{}{"gte": 99.5}},
			want:       true,
		},
		{
			name:       "float64 vs int eq",
			variables:  map[string]interface{}{"v": 50.0},
			conditions: map[string]interface{}{"v": map[string]interface{}{"eq": 50}},
			want:       true,
		},
		{
			name:       "string numeric vs int eq",
			variables:  map[string]interface{}{"v": "42"},
			conditions: map[string]interface{}{"v": map[string]interface{}{"eq": 42}},
			want:       true,
		},
		{
			name:       "string non-numeric eq fails",
			variables:  map[string]interface{}{"v": "abc"},
			conditions: map[string]interface{}{"v": map[string]interface{}{"eq": 0}},
			want:       false,
		},
		{
			name:       "min_ prefix uses numeric comparison",
			variables:  map[string]interface{}{"amount": 1500},
			conditions: map[string]interface{}{"min_amount": 1000},
			want:       true,
		},
		{
			name:       "max_ prefix uses numeric comparison",
			variables:  map[string]interface{}{"amount": 50},
			conditions: map[string]interface{}{"max_amount": 100},
			want:       true,
		},
		{
			name:       "min_ prefix parses numeric string and applies threshold",
			variables:  map[string]interface{}{"amount": "100"},
			conditions: map[string]interface{}{"min_amount": 50},
			want:       true,
		},
		{
			name:       "min_ prefix fails when amount is not numeric",
			variables:  map[string]interface{}{"amount": "abc"},
			conditions: map[string]interface{}{"min_amount": 1000},
			want:       false,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := svc.evaluateConditions(tt.conditions, tt.variables); got != tt.want {
				t.Fatalf("evaluateConditions(%s)=%v want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestRoutingLookupValue_Direct verifies routingLookupValue handles top-level
// keys and returns (nil,false) on missing keys.
func TestRoutingLookupValue_Direct(t *testing.T) {
	values := map[string]interface{}{"k": "v", "n": 5}
	if v, ok := routingLookupValue(values, "k"); !ok || v != "v" {
		t.Fatalf("expected lookup of k=v got %v,%v", v, ok)
	}
	if _, ok := routingLookupValue(values, "missing"); ok {
		t.Fatalf("missing key must return ok=false")
	}
}
