package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"itsm-backend/connector"
)

// Send 实现 connector.Connector.Send
// 飞书 send_msg 接受 receive_id_type（open_id/email/chat_id/user_id）
// 这里统一用 open_id，调用方在 Channel 字段传 open_id 即可
func (c *Client) Send(ctx context.Context, msg *connector.Message) error {
	if msg == nil {
		return fmt.Errorf("feishu: nil message")
	}
	if msg.Channel == "" {
		return fmt.Errorf("feishu: channel (receive_id) is required")
	}
	body := buildFeishuMessageBody(msg)
	var out struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			MessageID string `json:"message_id"`
		} `json:"data"`
	}
	// 飞书接口：POST /open-apis/im/v1/messages?receive_id_type=open_id[&uuid=...]
	// 幂等透传：msg.ID 是 delivery command 的 idempotency key。飞书 create message
	// 支持 uuid 去重（同 uuid 返回同一 message_id），覆盖「外部发送成功但本地
	// audit 写入失败后重试」场景，避免重试产生重复消息。
	if err := c.doJSON(ctx, "POST", feishuMessageSendPath("open_id", msg.ID), body, &out); err != nil {
		return err
	}
	if out.Code != 0 {
		return fmt.Errorf("feishu: send msg failed code=%d msg=%s", out.Code, out.Msg)
	}
	return nil
}

func buildFeishuMessageBody(msg *connector.Message) map[string]interface{} {
	content := msg.Content
	if msg.Card != nil {
		// 富文本卡片
		cardJSON, _ := json.Marshal(msg.Card)
		content = string(cardJSON)
		return map[string]interface{}{
			"receive_id": msg.Channel,
			"msg_type":   "interactive",
			"content":    content,
		}
	}
	switch msg.Type {
	case "markdown", "post":
		// post 类型：content 是 {zh_cn: {title, content: [[tag,text],...]}}
		if msg.Type == "post" {
			return map[string]interface{}{
				"receive_id": msg.Channel,
				"msg_type":   "post",
				"content":    content,
			}
		}
		// markdown 作为 post 发送
		post := map[string]interface{}{
			"zh_cn": map[string]interface{}{
				"title":   msg.Title,
				"content": [][]map[string]string{{{"tag": "text", "text": msg.Content}}},
			},
		}
		b, _ := json.Marshal(post)
		return map[string]interface{}{
			"receive_id": msg.Channel,
			"msg_type":   "post",
			"content":    string(b),
		}
	case "image":
		return map[string]interface{}{
			"receive_id": msg.Channel,
			"msg_type":   "image",
			"content":    content,
		}
	default:
		// text
		text, _ := json.Marshal(map[string]string{"text": msg.Content})
		return map[string]interface{}{
			"receive_id": msg.Channel,
			"msg_type":   "text",
			"content":    string(text),
		}
	}
}

// Reply 飞书"回复某条消息"接口
func (c *Client) Reply(ctx context.Context, messageID string, msg *connector.Message) error {
	body := buildFeishuMessageBody(msg)
	var out struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := c.doJSON(ctx, "POST", feishuMessageReplyPath(messageID, msg.ID), body, &out); err != nil {
		return err
	}
	if out.Code != 0 {
		return fmt.Errorf("feishu: reply failed code=%d msg=%s", out.Code, out.Msg)
	}
	return nil
}

// feishuUUID 把 delivery command 的 idempotency key 规范成飞书 uuid 参数：
// 去除首尾空白并防御性截断（飞书 uuid 上限 50 字节；真实 key 由 commandbus
// 生成，正常远短于该限制）。
func feishuUUID(idempotencyKey string) string {
	u := strings.TrimSpace(idempotencyKey)
	if len(u) > 50 {
		u = u[:50]
	}
	return u
}

func feishuMessageSendPath(receiveIDType, idempotencyKey string) string {
	path := "/open-apis/im/v1/messages?receive_id_type=" + url.QueryEscape(receiveIDType)
	if u := feishuUUID(idempotencyKey); u != "" {
		path += "&uuid=" + url.QueryEscape(u)
	}
	return path
}

func feishuMessageReplyPath(messageID, idempotencyKey string) string {
	path := "/open-apis/im/v1/messages/" + url.PathEscape(messageID) + "/reply"
	if u := feishuUUID(idempotencyKey); u != "" {
		path += "?uuid=" + url.QueryEscape(u)
	}
	return path
}
