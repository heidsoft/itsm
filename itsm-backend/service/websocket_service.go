package service

import (
	"encoding/json"
	"fmt"
	"itsm-backend/dto"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源（生产环境应该限制）
	},
}

// WebSocketMessage WebSocket消息
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// WebSocketClient WebSocket客户端
type WebSocketClient struct {
	ID        string
	UserID    int
	TenantID  int
	Conn      *websocket.Conn
	Send      chan []byte
	Hub       *WebSocketHub
	mu        sync.Mutex
	IsClosed  bool
}

// WebSocketHub WebSocket中心
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan []byte
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	logger     *zap.SugaredLogger
	mu         sync.RWMutex
}

// NewWebSocketHub 创建WebSocket中心
func NewWebSocketHub(logger *zap.SugaredLogger) *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		logger:     logger,
	}
}

// Run 运行WebSocket中心
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Infow("WebSocket client registered", "user_id", client.UserID, "tenant_id", client.TenantID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				h.logger.Infow("WebSocket client unregistered", "user_id", client.UserID)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// RegisterClient 注册客户端
func (h *WebSocketHub) RegisterClient(client *WebSocketClient) {
	h.register <- client
}

// UnregisterClient 注销客户端
func (h *WebSocketHub) UnregisterClient(client *WebSocketClient) {
	h.unregister <- client
}

// SendToUser 发送消息给指定用户
func (h *WebSocketHub) SendToUser(userID int, message WebSocketMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	msgBytes, err := json.Marshal(message)
	if err != nil {
		h.logger.Errorw("Failed to marshal websocket message", "error", err)
		return
	}

	for client := range h.clients {
		if client.UserID == userID && !client.IsClosed {
			select {
			case client.Send <- msgBytes:
			default:
				h.logger.Warnw("Failed to send message to user", "user_id", userID)
			}
		}
	}
}

// SendToTenant 发送消息给租户所有用户
func (h *WebSocketHub) SendToTenant(tenantID int, message WebSocketMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	msgBytes, err := json.Marshal(message)
	if err != nil {
		h.logger.Errorw("Failed to marshal websocket message", "error", err)
		return
	}

	for client := range h.clients {
		if client.TenantID == tenantID && !client.IsClosed {
			select {
			case client.Send <- msgBytes:
			default:
				h.logger.Warnw("Failed to send message to tenant user", "user_id", client.UserID)
			}
		}
	}
}

// BroadcastToAll 广播消息给所有用户
func (h *WebSocketHub) BroadcastToAll(message WebSocketMessage) {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return
	}
	h.broadcast <- msgBytes
}

// ReadPump 读取客户端消息
func (c *WebSocketClient) ReadPump() {
	defer func() {
		c.Hub.UnregisterClient(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.Errorw("WebSocket error", "error", err, "client_id", c.ID)
			}
			break
		}

		// 处理客户端消息
		var wsMsg WebSocketMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			c.Hub.logger.Warnw("Invalid WebSocket message", "error", err)
			continue
		}

		c.handleMessage(wsMsg)
	}
}

// WritePump 写入消息给客户端
func (c *WebSocketClient) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 添加队列中的消息
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 处理客户端消息
func (c *WebSocketClient) handleMessage(msg WebSocketMessage) {
	switch msg.Type {
	case "ping":
		response := WebSocketMessage{Type: "pong", Payload: nil}
		msgBytes, _ := json.Marshal(response)
		c.Send <- msgBytes
	case "subscribe":
		// 处理订阅
		c.Hub.logger.Infow("Client subscribed", "client_id", c.ID)
	case "unsubscribe":
		// 处理取消订阅
		c.Hub.logger.Infow("Client unsubscribed", "client_id", c.ID)
	}
}

// WebSocketService WebSocket服务
type WebSocketService struct {
	hub    *WebSocketHub
	logger *zap.SugaredLogger
}

// NewWebSocketService 创建WebSocket服务
func NewWebSocketService(logger *zap.SugaredLogger) *WebSocketService {
	hub := NewWebSocketHub(logger)
	go hub.Run()

	return &WebSocketService{
		hub:    hub,
		logger: logger,
	}
}

// GetHub 获取WebSocket中心
func (s *WebSocketService) GetHub() *WebSocketHub {
	return s.hub
}

// HandleWebSocket 处理WebSocket连接
func (s *WebSocketService) HandleWebSocket(w http.ResponseWriter, r *http.Request, userID, tenantID int) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Errorw("Failed to upgrade WebSocket", "error", err)
		return
	}

	client := &WebSocketClient{
		ID:       fmt.Sprintf("%d-%d", userID, time.Now().Unix()),
		UserID:   userID,
		TenantID: tenantID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Hub:      s.hub,
	}

	s.hub.RegisterClient(client)

	go client.WritePump()
	go client.ReadPump()
}

// NotifyTicketCreated 通知工单创建
func (s *WebSocketService) NotifyTicketCreated(tenantID int, ticket *dto.TicketResponse) {
	msg := WebSocketMessage{
		Type: "ticket_created",
		Payload: map[string]interface{}{
			"ticket": ticket,
		},
	}
	s.hub.SendToTenant(tenantID, msg)
}

// NotifyTicketUpdated 通知工单更新
func (s *WebSocketService) NotifyTicketUpdated(tenantID int, ticket *dto.TicketResponse, changedFields []string) {
	msg := WebSocketMessage{
		Type: "ticket_updated",
		Payload: map[string]interface{}{
			"ticket":         ticket,
			"changed_fields": changedFields,
		},
	}
	s.hub.SendToTenant(tenantID, msg)
}

// NotifyTicketAssigned 通知工单分配
func (s *WebSocketService) NotifyTicketAssigned(tenantID int, ticket *dto.TicketResponse, assigneeID int) {
	msg := WebSocketMessage{
		Type: "ticket_assigned",
		Payload: map[string]interface{}{
			"ticket":      ticket,
			"assignee_id": assigneeID,
		},
	}
	// 发送给创建者
	s.hub.SendToUser(ticket.RequesterID, msg)
	// 发送给被分配人
	if assigneeID != ticket.RequesterID {
		s.hub.SendToUser(assigneeID, msg)
	}
}

// NotifyTicketCommented 通知工单评论
func (s *WebSocketService) NotifyTicketCommented(tenantID int, ticketID int, comment *dto.TicketCommentResponse) {
	msg := WebSocketMessage{
		Type: "ticket_commented",
		Payload: map[string]interface{}{
			"ticket_id": ticketID,
			"comment":   comment,
		},
	}
	s.hub.SendToTenant(tenantID, msg)
}

// NotifySLABreached 通知SLA违反
func (s *WebSocketService) NotifySLABreached(tenantID int, ticket *dto.TicketResponse, slaName string) {
	msg := WebSocketMessage{
		Type: "sla_breached",
		Payload: map[string]interface{}{
			"ticket":   ticket,
			"sla_name": slaName,
		},
	}
	// 发送给相关人员
	if ticket.AssigneeID > 0 {
		s.hub.SendToUser(ticket.AssigneeID, msg)
	}
	s.hub.SendToUser(ticket.RequesterID, msg)
}

// NotifyWorkflowTask 通知工作流任务
func (s *WebSocketService) NotifyWorkflowTask(tenantID int, userID int, task map[string]interface{}) {
	msg := WebSocketMessage{
		Type:    "workflow_task",
		Payload: task,
	}
	s.hub.SendToUser(userID, msg)
}

// NotifyApprovalRequired 通知需要审批
func (s *WebSocketService) NotifyApprovalRequired(tenantID int, userID int, approvalInfo map[string]interface{}) {
	msg := WebSocketMessage{
		Type:    "approval_required",
		Payload: approvalInfo,
	}
	s.hub.SendToUser(userID, msg)
}
