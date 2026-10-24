package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"itsm-backend/common"
	"itsm-backend/ent/enttest"
)

// TestMSPMiddleware_Gate verifies that SetMSPEnabled(false) makes the
// /api/v1/msp/* route family answer with an explicit capability-disabled
// error instead of 404, even when no auth or DB state would otherwise block
// them. This is the private-deployment mode gate.
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
		// Without a real user the middleware will respond 401 inside, not the
		// gate's 503. We only care the gate did not short-circuit.
		assert.NotEqual(t, http.StatusServiceUnavailable, w.Code, "gate should not report capability disabled when enabled")
	})

	t.Run("disabled reports capability unavailable instead of 404", func(t *testing.T) {
		prev := mspEnabled
		SetMSPEnabled(false)
		defer SetMSPEnabled(prev)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/v1/msp/customers", nil)
		MSPMiddleware(client)(c)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.True(t, c.IsAborted())

		var body struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    struct {
				Capability string `json:"capability"`
				State      string `json:"state"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, common.ServiceUnavailableCode, body.Code)
		assert.Equal(t, "msp", body.Data.Capability)
		assert.Equal(t, "disabled", body.Data.State)
		assert.NotEmpty(t, body.Message)
	})
}
