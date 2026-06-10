package dingtalk

import (
	"context"
	"io"
	"net/http"
)

// httpNewRequest 提取出来便于测试 mock
var httpNewRequest = func(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, url, body)
}
