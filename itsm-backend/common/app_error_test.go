package common

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Conflict constructors encode two different meanings and must not be
// interchangeable:
//   - NewConflictError: duplicate resource, message derived from the noun
//     ("<resource> already exists").
//   - NewStateConflictError: state/precondition conflict, message is exactly the
//     caller's sentence.
//
// Regression: state conflicts used NewConflictError, so the client received
// "Invalid service request status transition already exists: failed -> delivered"
// / "CI关系 already exists" while the actual reason stayed inside Detail — a
// field RespondError never exposes.
func TestNewStateConflictErrorKeepsCallerMessage(t *testing.T) {
	err := NewStateConflictError("不允许该状态迁移（failed → delivered）", "failed -> delivered")

	assert.Equal(t, ErrCodeConflict, err.Code)
	assert.Equal(t, http.StatusConflict, err.HTTPStatus)
	assert.Equal(t, "不允许该状态迁移（failed → delivered）", err.Message)
	assert.Equal(t, "failed -> delivered", err.Detail)

	assert.False(t, strings.Contains(err.Message, "already exists"),
		"a state conflict must not borrow duplicate-resource wording")
	// The debug rendering keeps the structured suffix for logs...
	assert.Equal(t, "[CONFLICT] 不允许该状态迁移（failed → delivered）: failed -> delivered", err.Error())
}

func TestNewConflictErrorStaysDuplicateOnly(t *testing.T) {
	err := NewConflictError("ticket", "duplicate")
	assert.Equal(t, "ticket already exists", err.Message)
	assert.Equal(t, ErrCodeConflict, err.Code)
	assert.Equal(t, http.StatusConflict, err.HTTPStatus)
}
