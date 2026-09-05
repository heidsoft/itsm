package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/common"
	"itsm-backend/dto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockNotificationService struct{ mock.Mock }

func (m *mockNotificationService) CreateNotification(ctx context.Context, req *dto.CreateNotificationRequest) (*dto.Notification, error) {
	a := m.Called(ctx, req)
	if n, ok := a.Get(0).(*dto.Notification); ok {
		return n, a.Error(1)
	}
	return nil, a.Error(1)
}
func (m *mockNotificationService) GetNotifications(ctx context.Context, req *dto.GetNotificationsRequest) (*dto.NotificationListResponse, error) {
	a := m.Called(ctx, req)
	if n, ok := a.Get(0).(*dto.NotificationListResponse); ok {
		return n, a.Error(1)
	}
	return nil, a.Error(1)
}
func (m *mockNotificationService) MarkNotificationRead(ctx context.Context, req *dto.MarkNotificationReadRequest) error {
	return m.Called(ctx, req).Error(0)
}
func (m *mockNotificationService) MarkAllNotificationsRead(ctx context.Context, req *dto.MarkAllNotificationsReadRequest) error {
	return m.Called(ctx, req).Error(0)
}
func (m *mockNotificationService) DeleteNotification(ctx context.Context, req *dto.DeleteNotificationRequest) error {
	return m.Called(ctx, req).Error(0)
}
func (m *mockNotificationService) MarkNotificationsRead(ctx context.Context, notificationIDs []int, userID, tenantID int) (int, error) {
	a := m.Called(ctx, notificationIDs, userID, tenantID)
	return a.Int(0), a.Error(1)
}
func (m *mockNotificationService) DeleteNotifications(ctx context.Context, notificationIDs []int, userID, tenantID int) (int, error) {
	a := m.Called(ctx, notificationIDs, userID, tenantID)
	return a.Int(0), a.Error(1)
}
func (m *mockNotificationService) GetUnreadCount(ctx context.Context, userID, tenantID int) (int, error) {
	a := m.Called(ctx, userID, tenantID)
	return a.Int(0), a.Error(1)
}
func (m *mockNotificationService) GetCurrentUserID(c *gin.Context) (int, error) {
	a := m.Called(c)
	return a.Int(0), a.Error(1)
}
func (m *mockNotificationService) GetCurrentTenantID(c *gin.Context) (int, error) {
	a := m.Called(c)
	return a.Int(0), a.Error(1)
}

type mockNotificationPreferenceService struct{ mock.Mock }

func (m *mockNotificationPreferenceService) GetUserPreferences(ctx context.Context, userID, tenantID int) ([]*dto.NotificationPreferenceResponse, error) {
	a := m.Called(ctx, userID, tenantID)
	if l, ok := a.Get(0).([]*dto.NotificationPreferenceResponse); ok {
		return l, a.Error(1)
	}
	return nil, a.Error(1)
}
func (m *mockNotificationPreferenceService) GetUserPreferenceByEventType(ctx context.Context, userID, tenantID int, eventType string) (*dto.NotificationPreferenceResponse, error) {
	a := m.Called(ctx, userID, tenantID, eventType)
	if p, ok := a.Get(0).(*dto.NotificationPreferenceResponse); ok {
		return p, a.Error(1)
	}
	return nil, a.Error(1)
}
func (m *mockNotificationPreferenceService) CreateOrUpdatePreference(ctx context.Context, userID, tenantID int, req *dto.NotificationPreferenceRequest) (*dto.NotificationPreferenceResponse, error) {
	a := m.Called(ctx, userID, tenantID, req)
	if p, ok := a.Get(0).(*dto.NotificationPreferenceResponse); ok {
		return p, a.Error(1)
	}
	return nil, a.Error(1)
}
func (m *mockNotificationPreferenceService) BulkUpdatePreferences(ctx context.Context, userID, tenantID int, req *dto.BulkNotificationPreferenceRequest) ([]*dto.NotificationPreferenceResponse, error) {
	a := m.Called(ctx, userID, tenantID, req)
	if l, ok := a.Get(0).([]*dto.NotificationPreferenceResponse); ok {
		return l, a.Error(1)
	}
	return nil, a.Error(1)
}
func (m *mockNotificationPreferenceService) DeletePreference(ctx context.Context, userID, tenantID int, eventType string) error {
	return m.Called(ctx, userID, tenantID, eventType).Error(0)
}
func (m *mockNotificationPreferenceService) ResetToDefaults(ctx context.Context, userID, tenantID int) error {
	return m.Called(ctx, userID, tenantID).Error(0)
}
func (m *mockNotificationPreferenceService) InitializeDefaultPreferences(ctx context.Context, userID, tenantID int) error {
	return m.Called(ctx, userID, tenantID).Error(0)
}

func newNotificationHandler(n *mockNotificationService, p *mockNotificationPreferenceService) *Handler {
	gin.SetMode(gin.TestMode)
	return NewHandler(n, p, zap.NewNop().Sugar())
}

func notifCtx(method, path, body string) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if body != "" {
		c.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
	} else {
		c.Request = httptest.NewRequest(method, path, nil)
	}
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	return w, c
}

func TestGetNotifications_Success(t *testing.T) {
	n := &mockNotificationService{}
	h := newNotificationHandler(n, &mockNotificationPreferenceService{})
	n.On("GetCurrentUserID", mock.Anything).Return(7, nil)
	n.On("GetCurrentTenantID", mock.Anything).Return(3, nil)
	n.On("GetNotifications", mock.Anything, mock.Anything).Return(&dto.NotificationListResponse{
		Total: 1, Notifications: []dto.Notification{{ID: 1, Title: "hi"}},
	}, nil)

	w, c := notifCtx(http.MethodGet, "/api/v1/notifications", "")
	h.GetNotifications(c)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp common.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, common.SuccessCode, resp.Code)
	n.AssertExpectations(t)
}

func TestGetNotifications_Unauthenticated(t *testing.T) {
	n := &mockNotificationService{}
	h := newNotificationHandler(n, &mockNotificationPreferenceService{})
	n.On("GetCurrentUserID", mock.Anything).Return(0, assert.AnError)

	w, c := notifCtx(http.MethodGet, "/api/v1/notifications", "")
	h.GetNotifications(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	n.AssertNotCalled(t, "GetNotifications")
}

func TestGetUnreadCount_Success(t *testing.T) {
	n := &mockNotificationService{}
	h := newNotificationHandler(n, &mockNotificationPreferenceService{})
	n.On("GetCurrentUserID", mock.Anything).Return(7, nil)
	n.On("GetCurrentTenantID", mock.Anything).Return(3, nil)
	n.On("GetUnreadCount", mock.Anything, 7, 3).Return(5, nil)

	w, c := notifCtx(http.MethodGet, "/api/v1/notifications/unread-count", "")
	h.GetUnreadCount(c)

	assert.Equal(t, http.StatusOK, w.Code)
	n.AssertExpectations(t)
}

func TestMarkNotificationRead_Success(t *testing.T) {
	n := &mockNotificationService{}
	h := newNotificationHandler(n, &mockNotificationPreferenceService{})
	n.On("GetCurrentUserID", mock.Anything).Return(7, nil)
	n.On("GetCurrentTenantID", mock.Anything).Return(3, nil)
	n.On("MarkNotificationRead", mock.Anything, mock.Anything).Return(nil)

	w, c := notifCtx(http.MethodPost, "/api/v1/notifications/1/read", "")
	h.MarkNotificationRead(c)

	assert.Equal(t, http.StatusOK, w.Code)
	n.AssertExpectations(t)
}

func TestListPreferences_Success(t *testing.T) {
	n := &mockNotificationService{}
	p := &mockNotificationPreferenceService{}
	h := newNotificationHandler(n, p)
	p.On("GetUserPreferences", mock.Anything, 7, 3).Return([]*dto.NotificationPreferenceResponse{
		{EventType: "ticket.assign"},
	}, nil)

	w, c := notifCtx(http.MethodGet, "/api/v1/notifications/preferences", "")
	// ListPreferences 直接从上下文读 user_id/tenant_id（裸 key），不经 notificationService。
	c.Set("user_id", 7)
	c.Set("tenant_id", 3)
	h.ListPreferences(c)

	assert.Equal(t, http.StatusOK, w.Code)
	p.AssertExpectations(t)
}