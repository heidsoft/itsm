package service

import "testing"

func TestSanitizeSpreadsheetCell(t *testing.T) {
	tests := map[string]string{
		"=cmd|' /C calc'!A0": "'=cmd|' /C calc'!A0",
		" +SUM(A1:A2)":       "' +SUM(A1:A2)",
		"-1+2":               "'-1+2",
		"@SUM(A1:A2)":        "'@SUM(A1:A2)",
		"normal":             "normal",
		"":                   "",
	}
	for input, want := range tests {
		if got := sanitizeSpreadsheetCell(input); got != want {
			t.Errorf("sanitizeSpreadsheetCell(%q) = %q, want %q", input, got, want)
		}
	}
}
