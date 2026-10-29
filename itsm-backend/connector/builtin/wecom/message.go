package wecom

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"itsm-backend/connector"
)

// Send 实现 connector.Connector.Send
// channel 字段：
//   - "@all"                            -> 应用全员
//   - "UserID1|UserID2" 或 "UserID"     -> 指定用户
//   - "PartyID"                         -> 指定部门
//   - "TagID"                           -> 指定标签
//   - URL 开头（http/https）            -> 群机器人 webhook
//
// 幂等边界：企微应用消息/群机器人协议均无幂等参数（外部系统适配边界），
// msg.ID（delivery command idempotency key）无法透传；重试去重依赖
// commandbus 层的幂等与 delivery 状态检查，不在本适配器内伪装。
func (c *Client) Send(ctx context.Context, msg *connector.Message) error {
	if msg == nil || msg.Channel == "" {
		return fmt.Errorf("wecom: channel is required")
	}
	if strings.HasPrefix(msg.Channel, "http://") || strings.HasPrefix(msg.Channel, "https://") {
		return c.sendRobot(ctx, msg)
	}
	return c.sendApp(ctx, msg)
}

func (c *Client) sendApp(ctx context.Context, msg *connector.Message) error {
	tok, err := c.Token(ctx)
	if err != nil {
		return err
	}
	if c.agentID == "" {
		return fmt.Errorf("wecom: agent_id is required for app message")
	}
	payload := map[string]interface{}{
		"touser":  msg.Channel,
		"msgtype": mapWecomType(msg.Type),
		"agentid": mustInt(c.agentID),
	}
	switch msg.Type {
	case "markdown":
		body := map[string]interface{}{
			"content": buildMarkdown(msg),
		}
		// 企业微信 markdown 应用消息需要单独的 content 字段
		payload["markdown"] = body
	case "card", "textcard":
		btns := make([]map[string]string, 0)
		for _, a := range msg.Actions {
			btns = append(btns, map[string]string{"title": a.Text, "url": a.URL})
		}
		payload["textcard"] = map[string]interface{}{
			"title":       msg.Title,
			"description": msg.Content,
			"url":         ifEmpty(btnURL(btns), "https://example.com"),
			"btntxt":      ifEmpty(btnTitle(btns), "详情"),
		}
	default:
		payload["text"] = map[string]interface{}{"content": msg.Content}
	}
	var out struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := c.doJSON(ctx, "POST", "/cgi-bin/message/send",
		map[string]string{"access_token": tok}, payload, &out); err != nil {
		return err
	}
	if out.ErrCode != 0 {
		return fmt.Errorf("wecom: app message failed code=%d msg=%s", out.ErrCode, out.ErrMsg)
	}
	return nil
}

func (c *Client) sendRobot(ctx context.Context, msg *connector.Message) error {
	body := map[string]interface{}{"msgtype": mapWecomType(msg.Type)}
	switch msg.Type {
	case "markdown":
		body["markdown"] = map[string]interface{}{"content": buildMarkdown(msg)}
	case "card", "news":
		// picurl 为可选字段：metadata 缺失或 image 非字符串时按空串处理，绝不能用未检查的类型断言。
		picurl, _ := msg.Metadata["image"].(string)
		body["news"] = map[string]interface{}{
			"articles": []map[string]string{{
				"title":       msg.Title,
				"description": msg.Content,
				"url":         ifEmpty(firstActionURL(msg), "https://example.com"),
				"picurl":      picurl,
			}},
		}
	default:
		body["text"] = map[string]interface{}{
			"content":               msg.Content,
			"mentioned_list":        mentionedList(msg),
			"mentioned_mobile_list": mentionedMobile(msg),
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("wecom: marshal robot message: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", msg.Channel, strings.NewReader(string(raw)))
	if err != nil {
		return fmt.Errorf("wecom: build robot request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("wecom: robot send: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("wecom: read robot response: %w", err)
	}
	// 传输层结果必须校验：HTML 网关错误页或非 2xx 都不能被当成发送成功。
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("wecom: robot send http status=%d", resp.StatusCode)
	}
	var out struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	// 解析失败即失败：不得吞掉 unmarshal 错误让 errcode 保持 0 伪装成功（消息会静默丢失）。
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("wecom: robot response not JSON: %w", err)
	}
	if out.ErrCode != 0 {
		return fmt.Errorf("wecom: robot send failed code=%d msg=%s", out.ErrCode, out.ErrMsg)
	}
	return nil
}

func buildMarkdown(msg *connector.Message) string {
	var b strings.Builder
	if msg.Title != "" {
		b.WriteString("# ")
		b.WriteString(msg.Title)
		b.WriteString("\n")
	}
	b.WriteString(msg.Content)
	for _, a := range msg.Actions {
		b.WriteString("\n[")
		b.WriteString(a.Text)
		b.WriteString("](")
		b.WriteString(a.URL)
		b.WriteString(")")
	}
	return b.String()
}

func mapWecomType(t string) string {
	switch t {
	case "markdown":
		return "markdown"
	case "card", "textcard", "news":
		return t
	default:
		return "text"
	}
}

func mentionedList(msg *connector.Message) []string {
	out := make([]string, 0)
	for _, m := range msg.Mentions {
		if m.Type == "user" {
			out = append(out, m.ID)
		}
	}
	return out
}

func mentionedMobile(msg *connector.Message) []string {
	out := make([]string, 0)
	for _, m := range msg.Mentions {
		if m.Type == "mobile" {
			out = append(out, m.ID)
		}
	}
	return out
}

func ifEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func btnURL(btns []map[string]string) string {
	if len(btns) == 0 {
		return ""
	}
	return btns[0]["url"]
}

func btnTitle(btns []map[string]string) string {
	if len(btns) == 0 {
		return ""
	}
	return btns[0]["title"]
}

func firstActionURL(msg *connector.Message) string {
	if len(msg.Actions) == 0 {
		return ""
	}
	return msg.Actions[0].URL
}

func mustInt(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}
