package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"itsm-backend/connector"
	_ "itsm-backend/connector/builtin/console"
	_ "itsm-backend/connector/builtin/dingtalk"
	_ "itsm-backend/connector/builtin/email"
	_ "itsm-backend/connector/builtin/feishu"
	_ "itsm-backend/connector/builtin/webhook"
	_ "itsm-backend/connector/builtin/wecom"
	aiHandler "itsm-backend/handlers/ai"
	connectorHandler "itsm-backend/handlers/connector"
	notificationHandler "itsm-backend/handlers/notification"
	skillHandler "itsm-backend/handlers/skill"
	vectorHandler "itsm-backend/handlers/vector_store"
	"itsm-backend/service"
)

// Exercise the real list/status handlers with immutable built-in metadata and no database.
func TestBuiltinMetadataEnglishHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := zap.NewNop().Sugar()
	registry := service.NewSkillRegistry()
	require.NoError(t, aiHandler.RegisterBuiltinSkills(registry, &aiHandler.Service{}, logger))
	skills := skillHandler.NewHandler(registry, logger, nil)
	connectors := connectorHandler.NewHandler(connector.NewManager(connector.Default(), logger), connector.Default(), nil, logger)
	vector := vectorHandler.NewHandler(nil, logger)
	r := gin.New()
	r.GET("/api/v1/admin/skills", skills.List)
	r.GET("/api/v1/connectors", connectors.ListMarket)
	r.GET("/api/v1/system/vector-store", vector.GetStatus)
	r.GET("/api/v1/notification-preferences/event-types", (&notificationHandler.Handler{}).ListEventTypes)
	t.Setenv("VECTOR_STORE_CONFIG", "")
	for _, path := range []string{"/api/v1/admin/skills", "/api/v1/connectors", "/api/v1/system/vector-store", "/api/v1/notification-preferences/event-types"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
			require.Equal(t, http.StatusOK, w.Code)
			var envelope struct {
				Code int             `json:"code"`
				Data json.RawMessage `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
			require.Zero(t, envelope.Code)
			require.NotEmpty(t, envelope.Data)
			for _, ch := range string(envelope.Data) {
				require.False(t, unicode.Is(unicode.Han, ch), "fixed metadata must use English")
			}
			if path == "/api/v1/system/vector-store" {
				require.Contains(t, string(envelope.Data), `"capability":"unconfigured"`)
				require.Contains(t, string(envelope.Data), "keyword search fallback")
			} else if path == "/api/v1/notification-preferences/event-types" {
				require.Contains(t, string(envelope.Data), `"code":"ticket_created"`)
				require.Contains(t, string(envelope.Data), `"name":"Ticket Created"`)
			} else {
				require.Contains(t, string(envelope.Data), `"items":[{`)
			}
		})
	}
}
