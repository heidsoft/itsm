package aliyun

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 官方签名示例：https://help.aliyun.com/zh/sdk/product-overview/rpc-mechanism
func TestBuildRPCSignatureMatchesOfficialVector(t *testing.T) {
	params := map[string]string{
		"AccessKeyId":      "testid",
		"Action":           "DescribeRegions",
		"Format":           "XML",
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   "3ee8c1b8-83d3-44af-a94f-4e0ad82fd6cf",
		"SignatureVersion": "1.0",
		"Timestamp":        "2016-02-23T12:46:24Z",
		"Version":          "2014-05-26",
	}

	require.Equal(t, "OLeaidS1JvxuMvnyHOwuJ+uX5qY=", BuildRPCSignature("GET", params, "testsecret"))
}

func TestBuildRPCSignatureIgnoresMapOrder(t *testing.T) {
	first := map[string]string{"Action": "DescribeVulList", "Type": "cve", "StatusList": "1"}
	second := map[string]string{"StatusList": "1", "Type": "cve", "Action": "DescribeVulList"}

	require.Equal(t, BuildRPCSignature("POST", first, "secret"), BuildRPCSignature("POST", second, "secret"))
}

func TestBuildRPCSignatureEncodesReservedCharacters(t *testing.T) {
	plain := BuildRPCSignature("POST", map[string]string{"Dimensions": "a b"}, "secret")
	encoded := BuildRPCSignature("POST", map[string]string{"Dimensions": "a+b"}, "secret")

	require.NotEqual(t, plain, encoded, "空格与加号必须编码成不同的值")
}

func TestParseRPCErrorKeepsProviderCode(t *testing.T) {
	err := parseRPCError(403, []byte(`{"Code":"Forbidden.RAM","Message":"User not authorized","RequestId":"req-1"}`))

	rpcErr, ok := err.(*RPCError)
	require.True(t, ok)
	require.Equal(t, 403, rpcErr.StatusCode)
	require.Equal(t, "Forbidden.RAM", rpcErr.Code)
	require.Equal(t, "req-1", rpcErr.RequestID)
}

// TestCallRejectsHTTP200BusinessError 守护「HTTP 200 + 业务错误码」这条盲区。
//
// 云安全中心被限流时返回的是 200 + {"Code":"Throttling"}：此前只判状态码，
// 响应体会被解成全零值，上层据此认为「云上一条漏洞都没有」，最终把存量
// 未修复漏洞全部误标已修复。这条用例保证每个调用点都拿得到 error。
func TestCallRejectsHTTP200BusinessError(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantCode string
	}{
		{name: "throttling", body: `{"Code":"Throttling","Message":"Request was denied due to flow control.","RequestId":"req-throttle"}`, wantCode: "Throttling"},
		{name: "forbidden", body: `{"Code":"Forbidden.RAM","Message":"User not authorized"}`, wantCode: "Forbidden.RAM"},
		{name: "numeric code", body: `{"Code":403,"Message":"no permission"}`, wantCode: "403"},
		{name: "success false without code", body: `{"Success":false,"Message":"internal"}`, wantCode: "UnknownError"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestRPCClient(t, http.StatusOK, tc.body)
			var out SecurityCenterVulStatistics
			err := client.Call(context.Background(), "DescribeVulNumStatistics", nil, &out)

			var rpcErr *RPCError
			require.ErrorAs(t, err, &rpcErr)
			require.Equal(t, tc.wantCode, rpcErr.Code)
			require.Equal(t, http.StatusOK, rpcErr.StatusCode)
			// 关键：不能把错误体解成「零值数据」交给上层。
			require.Equal(t, 0, out.TotalUnfixed())
		})
	}
}

// TestCallAcceptsGenuineSuccess 确认收紧判定后没有误伤正常响应。
func TestCallAcceptsGenuineSuccess(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "data only", body: `{"CveNum":2,"SysNum":1,"EmgNum":3,"VulAsapSum":6,"RequestId":"req-ok"}`},
		{name: "cms success code 200", body: `{"Code":"200","Success":true,"Total":1,"AlertLogList":[]}`},
		{name: "cms success code 0", body: `{"Code":0,"Success":true,"Total":0}`},
		{name: "ok marker", body: `{"Code":"OK","Total":0}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestRPCClient(t, http.StatusOK, tc.body)
			var out json.RawMessage
			require.NoError(t, client.Call(context.Background(), "DescribeVulNumStatistics", nil, &out))
		})
	}
}

// TestCallStillRejectsNon200 确认原有状态码分支没有被新闸门取代。
func TestCallStillRejectsNon200(t *testing.T) {
	client := newTestRPCClient(t, http.StatusForbidden, `{"Code":"Forbidden.RAM","Message":"denied"}`)
	var out SecurityCenterVulStatistics

	var rpcErr *RPCError
	require.ErrorAs(t, client.Call(context.Background(), "DescribeVulList", nil, &out), &rpcErr)
	require.Equal(t, http.StatusForbidden, rpcErr.StatusCode)
}

// newTestRPCClient 构造一个指向本地假服务端的 rpcClient。
//
// 复用生产 callRaw 的完整签名与编码路径（只替换 endpoint 与 TLS 根证书），
// 因此这里验的就是生产链路：请求真的被发出、响应真的按生产规则判定。
// callRaw 固定拼 https://，所以假服务端也用 TLS；用 server.Client() 拿到
// 它自签证书的信任链。
func newTestRPCClient(t *testing.T, status int, body string) *rpcClient {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	client := newRPCClient(strings.TrimPrefix(server.URL, "https://"), "2018-12-03", "ak", "sk", "")
	client.httpClient = server.Client()
	return client
}
