package dingtalk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"itsm-backend/connector"
)

// callSendRobot 把被测机器人发送包在 recover 里：修复前忽略 NewRequest 错误后会在 nil 请求上解引用 panic。
func callSendRobot(t *testing.T, c *Client, msg *connector.Message) (err error) {
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

	// appSecret 为空以跳过加签，隔离 URL/状态码行为。
	c := NewClient("ak", "", "agent", "")
	err := callSendRobot(t, c, &connector.Message{Channel: srv.URL, Type: "text", Content: "hello"})
	if err != nil {
		t.Fatalf("合法 200 + errcode:0 应成功，得到 err=%v", err)
	}
	if !hit {
		t.Fatal("请求应到达机器人 webhook")
	}
}

func TestSendRobot_BadURLReturnsError(t *testing.T) {
	c := NewClient("ak", "", "agent", "")
	err := callSendRobot(t, c, &connector.Message{Channel: "https://a%zz.com", Type: "text", Content: "x"})
	if err == nil {
		t.Fatal("非法 webhook URL 必须返回 error，而不是在 nil 请求上解引用 panic")
	}
}

func TestSendRobot_HTTPErrorStatusIsNotSuccess(t *testing.T) {
	// 钉钉机器人此前只校验 unmarshal/errcode，未校验 HTTP status：500 + errcode:0 会被当成功。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer srv.Close()

	c := NewClient("ak", "", "agent", "")
	err := callSendRobot(t, c, &connector.Message{Channel: srv.URL, Type: "text", Content: "x"})
	if err == nil {
		t.Fatal("HTTP 500 必须返回 error")
	}
}
