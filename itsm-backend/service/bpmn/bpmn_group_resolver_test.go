package bpmn

import (
	"reflect"
	"testing"
)

func TestSplitCSV(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"single", "managers", []string{"managers"}},
		{"multiple", "managers,engineers,ops", []string{"managers", "engineers", "ops"}},
		{"spaces", " managers , engineers ,  ops  ", []string{"managers", "engineers", "ops"}},
		{"trailing-comma", "managers,", []string{"managers"}},
		{"empty-parts", "managers,,, engineers", []string{"managers", "engineers"}},
		{"only-commas", ",,,", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := splitCSV(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("splitCSV(%q) = %#v, want %#v", tc.in, got, tc.want)
			}
		})
	}
}

func TestJoinCSV(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want string
	}{
		{"nil", nil, ""},
		{"empty", []string{}, ""},
		{"single", []string{"managers"}, "managers"},
		{"dedup", []string{"managers", "engineers", "managers"}, "managers,engineers"},
		{"trim", []string{" managers ", "  engineers  "}, "managers,engineers"},
		{"empty-item", []string{"managers", "", "engineers"}, "managers,engineers"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := joinCSV(tc.in)
			if got != tc.want {
				t.Errorf("joinCSV(%#v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestMergeCandidateUsers(t *testing.T) {
	r := &GroupResolver{}
	cases := []struct {
		name      string
		bpmn      string
		fromGroup []string
		want      string
	}{
		{"both-empty", "", nil, ""},
		{"only-bpmn", "alice,bob", nil, "alice,bob"},
		{"only-group", "", []string{"carol", "dave"}, "carol,dave"},
		{"merged-dedup", "alice,bob", []string{"bob", "carol"}, "alice,bob,carol"},
		{"trim-spaces", "alice, bob", []string{" carol "}, "alice,bob,carol"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := r.MergeCandidateUsers(tc.bpmn, tc.fromGroup)
			if got != tc.want {
				t.Errorf("MergeCandidateUsers(%q, %#v) = %q, want %q", tc.bpmn, tc.fromGroup, got, tc.want)
			}
		})
	}
}
