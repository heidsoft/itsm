package connector

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// InboundHandler 入站消息处理器：连接器收到 IM 回调后调用
// 内部以 Channel 为 key 做去重幂等（避免 IM 平台重试造成重复消息）
type InboundHandler func(ctx context.Context, msg *InboundMessage) error

// Router 负责把"入站消息"分发到"业务处理器"
// 典型用法：
//   1. 飞书事件回调 -> 入站消息 -> 创建/查询工单 -> 回发卡片
//   2. 钉钉机器人对话 -> 识别意图 -> 调 AI -> 回复用户
type Router struct {
	logger *zap.SugaredLogger

	mu       sync.RWMutex
	handlers []InboundHandler
	seen     sync.Map // key = messageID + channel -> time.Time
	dedupTTL time.Duration
}

func NewRouter(logger *zap.SugaredLogger) *Router {
	return &Router{logger: logger, dedupTTL: 5 * time.Minute}
}

// Register 注册处理器（按注册顺序串行调用，第一个 error 即返回）
func (r *Router) Register(h InboundHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = append(r.handlers, h)
}

// Dispatch 派发入站消息
func (r *Router) Dispatch(ctx context.Context, msg *InboundMessage) error {
	if msg == nil {
		return nil
	}
	// 幂等去重
	key := msg.MessageID + "@" + msg.Channel
	if msg.MessageID != "" {
		if v, ok := r.seen.Load(key); ok {
			if t, ok := v.(time.Time); ok && time.Since(t) < r.dedupTTL {
				if r.logger != nil {
					r.logger.Debugw("duplicate inbound message skipped", "key", key)
				}
				return nil
			}
		}
		r.seen.Store(key, time.Now())
	}
	r.mu.RLock()
	handlers := append([]InboundHandler(nil), r.handlers...)
	r.mu.RUnlock()
	for i, h := range handlers {
		if err := h(ctx, msg); err != nil && r.logger != nil {
			r.logger.Errorw("inbound handler failed",
				"idx", i, "err", err, "channel", msg.Channel, "user", msg.UserID)
		}
	}
	return nil
}

// HTTPIngress 一个开箱即用的"入站网关"辅助函数：把 HTTP POST 解析为 InboundMessage 后 Dispatch
// 业务方可自行校验签名/解析负载后再调用 Dispatch
func (r *Router) HTTPIngress(maxBytes int64) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body := http.MaxBytesReader(w, req.Body, maxBytes)
		defer body.Close()
		var raw json.RawMessage
		if err := json.NewDecoder(body).Decode(&raw); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		// 默认把整个 payload 当 InboundMessage（具体连接器可在自己的 HTTP 入口自行 Parse）
		msg := &InboundMessage{
			Type:       "raw",
			Content:    string(raw),
			Raw:        raw,
			ReceivedAt: time.Now(),
		}
		_ = r.Dispatch(req.Context(), msg)
		w.WriteHeader(http.StatusOK)
	}
}
