package service

import "strings"

// sanitizeSpreadsheetCell prevents CSV/Excel formula injection when exported
// data is opened by spreadsheet software. The leading apostrophe is displayed
// as a text marker by common spreadsheet applications.
func sanitizeSpreadsheetCell(value string) string {
	trimmed := strings.TrimLeft(value, "\t\r\n ")
	if trimmed == "" {
		return value
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		return "'" + value
	default:
		return value
	}
}

func sanitizeSpreadsheetRow(row []string) []string {
	result := make([]string, len(row))
	for i, value := range row {
		result[i] = sanitizeSpreadsheetCell(value)
	}
	return result
}
