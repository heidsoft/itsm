package bpmn

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestIncidentServiceTaskHandler_UnknownAction(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:incident_unknown?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	handler := NewIncidentServiceTaskHandler(client, logger)

	result, err := handler.Execute(context.Background(), nil, map[string]interface{}{
		"action": "nonexistent_action",
	})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "未知的服务任务动作")
}

func TestChangeServiceTaskHandler_UnknownAction(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:change_unknown?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	handler := NewChangeServiceTaskHandler(client, logger)

	result, err := handler.Execute(context.Background(), nil, map[string]interface{}{
		"action": "nonexistent_action",
	})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "未知的服务任务动作")
}

func TestServiceRequestServiceTaskHandler_UnknownAction(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:sr_unknown?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	handler := NewServiceRequestServiceTaskHandler(client, logger)

	result, err := handler.Execute(context.Background(), nil, map[string]interface{}{
		"action": "nonexistent_action",
	})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "未知的服务任务动作")
}

func TestTicketServiceTaskHandler_UnknownAction(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_unknown?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	logger := zaptest.NewLogger(t).Sugar()
	handler := NewTicketServiceTaskHandler(client, logger)

	result, err := handler.Execute(context.Background(), nil, map[string]interface{}{
		"action":      "nonexistent_action",
		"business_id": 1,
	})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "未知的服务任务动作")
}
