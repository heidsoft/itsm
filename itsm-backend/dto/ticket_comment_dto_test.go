package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateTicketCommentRequestNormalizeLegacyField(t *testing.T) {
	legacyValue := true
	req := CreateTicketCommentRequest{
		Content:          "hello",
		IsInternalLegacy: &legacyValue,
	}

	req.Normalize()

	assert.True(t, req.IsInternal)
}

func TestUpdateTicketCommentRequestNormalizePrefersCamelCase(t *testing.T) {
	camelValue := false
	legacyValue := true
	req := UpdateTicketCommentRequest{
		IsInternal:       &camelValue,
		IsInternalLegacy: &legacyValue,
	}

	req.Normalize()

	if assert.NotNil(t, req.IsInternal) {
		assert.False(t, *req.IsInternal)
	}
}

func TestUpdateTicketCommentRequestNormalizeUsesLegacyWhenCamelMissing(t *testing.T) {
	legacyValue := true
	req := UpdateTicketCommentRequest{
		IsInternalLegacy: &legacyValue,
	}

	req.Normalize()

	if assert.NotNil(t, req.IsInternal) {
		assert.True(t, *req.IsInternal)
	}
}
