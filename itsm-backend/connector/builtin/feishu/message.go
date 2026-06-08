package feishu

import (
	"context"
	"encoding/json"
	"fmt"

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
	// 飞书接口：POST /open-apis/im/v1/messages?receive_id_type=open_id
	if err := c.doJSON(ctx, "POST", "/open-apis/im/v1/messages?receive_id_type=open_id", body, &out); err != nil {
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
				"content": [][]map[string]string{{{ "tag": "text", "text": msg.Content }}},
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
	if err := c.doJSON(ctx, "POST", "/open-apis/im/v1/messages/"+messageID+"/reply", body, &out); err != nil {
		return err
	}
	if out.Code != 0 {
		return fmt.Errorf("feishu: reply failed code=%d msg=%s", out.Code, out.Msg)
	}
	return nil
}
