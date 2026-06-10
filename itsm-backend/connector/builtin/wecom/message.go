package wecom

import (
	"context"
	"encoding/json"
	"fmt"
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
		body["news"] = map[string]interface{}{
			"articles": []map[string]string{{
				"title":       msg.Title,
				"description": msg.Content,
				"url":         ifEmpty(firstActionURL(msg), "https://example.com"),
				"picurl":      msg.Metadata["image"].(string),
			}},
		}
	default:
		body["text"] = map[string]interface{}{
			"content":             msg.Content,
			"mentioned_list":      mentionedList(msg),
			"mentioned_mobile_list": mentionedMobile(msg),
		}
	}
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", msg.Channel, strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	buf := make([]byte, 1024)
	n, _ := resp.Body.Read(buf)
	var out struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	_ = json.Unmarshal(buf[:n], &out)
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
		if c < '0' || c > '9' { break }
		n = n*10 + int(c-'0')
	}
	return n
}
