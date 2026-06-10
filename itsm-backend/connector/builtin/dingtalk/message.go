package dingtalk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"itsm-backend/connector"
)

// Send 实现 connector.Connector.Send
// channel 字段含义：
//   - "user:USERID" 或 "USERID"   -> 工作通知（发给指定 userid）
//   - "dept:DEPTID"               -> 部门通知
//   - "all"                       -> 全员
//   - 其它：当作群机器人 webhook URL
func (c *Client) Send(ctx context.Context, msg *connector.Message) error {
	if msg == nil || msg.Channel == "" {
		return fmt.Errorf("dingtalk: channel is required")
	}
	if strings.HasPrefix(msg.Channel, "http://") || strings.HasPrefix(msg.Channel, "https://") {
		return c.sendRobot(ctx, msg)
	}
	return c.sendWorkNotice(ctx, msg)
}

func (c *Client) sendWorkNotice(ctx context.Context, msg *connector.Message) error {
	tok, err := c.Token(ctx)
	if err != nil {
		return err
	}
	touser := strings.TrimPrefix(msg.Channel, "user:")
	touser = strings.TrimPrefix(touser, "dept:")
	deptID := ""
	if strings.HasPrefix(msg.Channel, "dept:") {
		touser = ""
		deptID = strings.TrimPrefix(msg.Channel, "dept:")
	}
	if c.agentID == "" {
		return fmt.Errorf("dingtalk: agent_id is required for work notice")
	}
	payload := map[string]interface{}{
		"agent_id":    c.agentID,
		"userid_list": strings.Split(touser, ","),
		"to_all_user": msg.Channel == "all",
		"msg":         buildDingTalkMsg(msg),
	}
	if deptID != "" {
		payload["dept_id_list"] = strings.Split(deptID, ",")
	}
	var out struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		TaskID  int64  `json:"task_id"`
	}
	if err := c.doJSON(ctx, "POST", "/topapi/message/corpconversation/asyncsend_v2",
		map[string]string{"access_token": tok}, payload, &out); err != nil {
		return err
	}
	if out.ErrCode != 0 {
		return fmt.Errorf("dingtalk: work notice failed code=%d msg=%s", out.ErrCode, out.ErrMsg)
	}
	return nil
}

func (c *Client) sendRobot(ctx context.Context, msg *connector.Message) error {
	webhook := msg.Channel
	// 加签：如果 settings 中传了 sign_secret，则签名
	if c.appSecret != "" {
		signed, err := SignRobotWebhook(webhook, c.appSecret)
		if err == nil {
			webhook = signed
		}
	}
	body := map[string]interface{}{
		"msgtype":  mapMsgType(msg.Type),
		"msgtype_": nil,
	}
	body[mapMsgType(msg.Type)] = buildRobotContent(msg)
	raw, _ := json.Marshal(body)
	req, _ := httpNewRequest(ctx, "POST", webhook, strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
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
	if err := json.Unmarshal(buf[:n], &out); err != nil {
		return fmt.Errorf("dingtalk: robot response: %w", err)
	}
	if out.ErrCode != 0 {
		return fmt.Errorf("dingtalk: robot send failed code=%d msg=%s", out.ErrCode, out.ErrMsg)
	}
	return nil
}

func buildDingTalkMsg(msg *connector.Message) map[string]interface{} {
	switch msg.Type {
	case "markdown":
		return map[string]interface{}{
			"msgtype": "markdown",
			"markdown": map[string]interface{}{
				"title": msg.Title,
				"text":  msg.Content,
			},
		}
	case "card", "actionCard":
		btns := make([]map[string]string, 0)
		for _, a := range msg.Actions {
			btns = append(btns, map[string]string{"title": a.Text, "actionURL": a.URL})
		}
		return map[string]interface{}{
			"msgtype": "actionCard",
			"actionCard": map[string]interface{}{
				"title":       msg.Title,
				"text":        msg.Content,
				"singleTitle": ifEmpty(firstBtn(btns, "title"), "查看详情"),
				"singleURL":   ifEmpty(firstBtn(btns, "actionURL"), "https://example.com"),
				"btns":        btns,
			},
		}
	default:
		return map[string]interface{}{
			"msgtype": "text",
			"text":    map[string]string{"content": msg.Content},
		}
	}
}

func buildRobotContent(msg *connector.Message) map[string]interface{} {
	return buildDingTalkMsg(msg)
}

func mapMsgType(t string) string {
	switch t {
	case "markdown":
		return "markdown"
	case "card", "actionCard":
		return "actionCard"
	default:
		return "text"
	}
}

func ifEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func firstBtn(btns []map[string]string, key string) string {
	if len(btns) == 0 {
		return ""
	}
	return btns[0][key]
}
