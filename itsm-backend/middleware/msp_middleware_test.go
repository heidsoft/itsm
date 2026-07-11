package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/ent/enttest"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestMSPMiddleware_Gate verifies that SetMSPEnabled(false) makes the
// /api/v1/msp/* routes return 404 even when no auth or DB state would
// otherwise block them. This is the private-deployment mode gate.
func TestMSPMiddleware_Gate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client := enttest.Open(t, "sqlite3", "file:msp_gate?mode=memory&_fk=1")
	defer client.Close()

	// enabled by default — middleware should NOT abort on the gate.
	t.Run("enabled passes through", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/msp/customers", nil)
		MSPMiddleware(client)(c)
		// Without a real user the middleware will respond 401 inside, not 404
		// from the gate. We only care the gate did not short-circuit.
		assert.NotEqual(t, http.StatusNotFound, w.Code, "gate should not 404 when enabled")
	})

	t.Run("disabled returns 404", func(t *testing.T) {
		prev := mspEnabled
		SetMSPEnabled(false)
		defer SetMSPEnabled(prev)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/msp/customers", nil)
		MSPMiddleware(client)(c)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.True(t, c.IsAborted())
		assert.Contains(t, w.Body.String(), "MSP routes are disabled")
	})
}
