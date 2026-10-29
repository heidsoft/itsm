package wecom

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/connector"
)

// callSend 把被测发送包在 recover 里：修复前遇到未检查的类型断言 / nil 请求会 panic，
// 这里转成一次干净的测试失败；修复后应正常返回 error 或 nil。
func callSend(t *testing.T, c *Client, msg *connector.Message) (err error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Send 不应 panic，但收到 panic: %v", r)
		}
	}()
	return c.Send(context.Background(), msg)
}

func TestSendRobot_Success(t *testing.T) {
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hit = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer srv.Close()

	c := NewClient("corp", "secret", "agent", "")
	err := callSend(t, c, &connector.Message{Channel: srv.URL, Type: "text", Content: "hello"})
	if err != nil {
		t.Fatalf("合法 200 + errcode:0 应成功，得到 err=%v", err)
	}
	if !hit {
		t.Fatal("请求应到达机器人 webhook")
	}
}

func TestSendRobot_MissingImageDoesNotPanic(t *testing.T) {
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hit = true
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer srv.Close()

	c := NewClient("corp", "secret", "agent", "")
	// news 卡片未提供 metadata.image：picurl 是可选字段，缺失/类型不符必须按空串处理，不得 panic。
	err := callSend(t, c, &connector.Message{
		Channel: srv.URL,
		Type:    "news",
		Title:   "t",
		Content: "c",
	})
	if err != nil {
		t.Fatalf("缺失可选 image 不应导致失败，得到 err=%v", err)
	}
	if !hit {
		t.Fatal("请求应到达机器人 webhook")
	}
}

func TestSendRobot_BadURLReturnsError(t *testing.T) {
	c := NewClient("corp", "secret", "agent", "")
	// 非法 URL（无效百分号转义）必须返回 error，而不是忽略 NewRequest 错误后在 nil 请求上解引用 panic。
	err := callSend(t, c, &connector.Message{Channel: "https://a%zz.com", Type: "text", Content: "x"})
	if err == nil {
		t.Fatal("非法 webhook URL 必须返回 error")
	}
}

func TestSendRobot_NonJSONBodyIsNotSuccess(t *testing.T) {
	// 网关返回 200 但正文是 HTML（非 JSON）：旧实现吞掉 unmarshal 错误、errcode 保持 0，
	// 会把“发送失败”伪装成“已发送”，消息静默丢失。必须 fail-closed 返回 error。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html>upstream error</html>`))
	}))
	defer srv.Close()

	c := NewClient("corp", "secret", "agent", "")
	err := callSend(t, c, &connector.Message{Channel: srv.URL, Type: "text", Content: "x"})
	if err == nil {
		t.Fatal("非 JSON 响应不得被当作发送成功")
	}
}

func TestSendRobot_HTTPErrorStatusIsNotSuccess(t *testing.T) {
	// HTTP 500 即使正文里 errcode=0，也必须判定失败：出站调用不能只看业务码忽略传输层结果。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer srv.Close()

	c := NewClient("corp", "secret", "agent", "")
	err := callSend(t, c, &connector.Message{Channel: srv.URL, Type: "text", Content: "x"})
	if err == nil {
		t.Fatal("HTTP 500 必须返回 error")
	}
}
