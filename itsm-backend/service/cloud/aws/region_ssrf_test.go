package aws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/service/cloud"
)

// TestValidateAWSRegion 锁住 C2 的字符集白名单：region 会被直接拼进请求
// host，任何超出 [a-z0-9-] 的字符都可能把带 SigV4 Authorization（含 AK ID）
// 与 X-Amz-Security-Token 的请求导向攻击者域名或 IMDS。
func TestValidateAWSRegion(t *testing.T) {
	valid := []string{
		"us-east-1", "cn-north-1", "ap-southeast-1", "eu-central-1",
		"us-gov-west-1", "a",
	}
	for _, region := range valid {
		assert.NoError(t, validateAWSRegion(region), "合法 region %q 必须放行", region)
	}

	// 每条都是真实可造成 host 混淆/凭据外带的形态，不是随手造的边界值。
	malicious := []string{
		"",                       // 空 region → https://ec2..amazonaws.com
		"evil.com/",              // 拼成 https://ec2.evil.com/.amazonaws.com
		"x@169.254.169.254/",     // userinfo 形式指向 IMDS
		"us-east-1.evil.com",     // 子域外带
		"US-EAST-1",              // 大写（host 大小写不敏感，但拼接与签名不一致）
		"us-east-1:443",          // 端口注入
		"us-east-1#@evil.com",    // fragment 截断
		"us_east_1",              // 下划线
		"-us-east-1",             // 以连字符开头
		"us-east-1-",             // 以连字符结尾
		"us--east-1",             // 畸形 label
		"us east 1",              // 空白
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // 64 字符，超 label 上限
	}
	for _, region := range malicious {
		require.Error(t, validateAWSRegion(region), "恶意 region %q 必须被拒绝", region)
	}
}

// TestAWSCall_RejectsMaliciousRegionBeforeSigning 验证出网点兜底：即使调用方
// 绕过 InitClients 直接构造 apiClient，也不会把带凭据的请求发出去。
func TestAWSCall_RejectsMaliciousRegionBeforeSigning(t *testing.T) {
	var received int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&received, 1)
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?><DescribeRegionsResponse><regionInfo/></DescribeRegionsResponse>`))
	}))
	defer srv.Close()

	for _, region := range []string{"evil.com/", "x@169.254.169.254/", "us-east-1.evil.com"} {
		cred, err := cloud.ResolveAWSCredential(context.Background(), testCred)
		require.NoError(t, err)
		// 不设 Endpoint：走默认端点推导，正是 region 拼 host 的路径。
		c := &apiClient{region: region, cred: cred, hc: srv.Client()}
		var out describeRegionsResponse
		err = c.call(context.Background(), "DescribeRegions", nil, &out)
		require.Error(t, err, "region %q 必须在签名前被拒绝", region)
		assert.Contains(t, err.Error(), "invalid region")
	}
	assert.EqualValues(t, 0, atomic.LoadInt32(&received), "一个字节的凭据都不该发到外部")
}

// TestAWSInitClients_RejectsMaliciousRegionEarly 验证装配层早失败：
// 非法 region 不该等到第一次 call 才被发现。
func TestAWSInitClients_RejectsMaliciousRegionEarly(t *testing.T) {
	a := NewAdapter(nil)
	_, err := a.InitClients(context.Background(), testAccount(testCred), []string{"us-east-1", "evil.com/"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid region")

	// 合法清单仍可装配，不能因为加了校验把正常路径打断。
	clients, err := a.InitClients(context.Background(), testAccount(testCred), []string{"us-east-1", "cn-north-1"})
	require.NoError(t, err)
	assert.Len(t, clients, 2)
}
