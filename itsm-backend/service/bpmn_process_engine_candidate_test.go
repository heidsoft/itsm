package service

import "testing"

func TestValidateCandidateDefinition(t *testing.T) {
	valid := map[string]interface{}{
		"domain":           "expense",
		"formSchema":       map[string]interface{}{"fields": []interface{}{map[string]interface{}{"name": "amount", "type": "number"}}},
		"approvalPolicy":   map[string]interface{}{"rules": []interface{}{map[string]interface{}{"when": "amount >= 5000", "role": "finance_manager"}}},
		"ontologyBindings": map[string]interface{}{}, "slaConfig": map[string]interface{}{},
	}
	if err := validateCandidateDefinition(valid); err != nil {
		t.Fatalf("valid candidate rejected: %v", err)
	}
	for name, candidate := range map[string]interface{}{
		"missing section": map[string]interface{}{"domain": "expense"},
		"missing fields":  map[string]interface{}{"domain": "expense", "formSchema": map[string]interface{}{}, "approvalPolicy": map[string]interface{}{}, "ontologyBindings": map[string]interface{}{}, "slaConfig": map[string]interface{}{}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateCandidateDefinition(candidate.(map[string]interface{})); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
