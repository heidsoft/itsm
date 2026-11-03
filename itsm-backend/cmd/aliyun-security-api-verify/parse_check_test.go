package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"itsm-backend/service/cloud/aliyun"
)

// TestRealAccountEMGFindingParsesCorrectly 用真实账号抓到的原始响应，
// 走生产解析路径，确认那条 emg 漏洞能被正确解析成可入库的记录。
//
// 这条夹具来自 2026-11-03 真实账号（cn-beijing）的 DescribeVulList 原始响应，
// 刻意保留真实的毫秒时间戳与数字型 PrimaryId——这两处正是此前出错的地方。
func TestRealAccountEMGFindingParsesCorrectly(t *testing.T) {
	const raw = `{
	  "TotalCount": 1,
	  "RequestId": "01A1A12A-361C-5047-8E21-F072AB864618",
	  "PageSize": 100,
	  "VulRecords": [{
	    "PrimaryId": 8218810662,
	    "Status": 1,
	    "Name": "emg:SCA:AVD-2026-1860246",
	    "AliasName": "OpenClaw 多个安全漏洞合集【版本对比】",
	    "Necessity": "asap",
	    "InstanceId": "i-2zeb9hx05c68a5tlazqm",
	    "InstanceName": "itsm-verify",
	    "RegionId": "cn-beijing",
	    "Uuid": "0f4a1c2d-1111-2222-3333-444455556666",
	    "InternetIp": "39.97.54.216",
	    "IntranetIp": "172.26.58.17",
	    "FirstTs": 1781413840000,
	    "LastTs": 1781413840000,
	    "ModifyTs": 1781413840000,
	    "ExtendContentJson": "{\"cvss_score\":\"8.8\",\"Necessity\":{}}"
	  }],
	  "CurrentPage": 1
	}`

	var page struct {
		TotalCount int                            `json:"TotalCount"`
		VulRecords []aliyun.SecurityVulnerability `json:"VulRecords"`
	}
	require.NoError(t, json.Unmarshal([]byte(raw), &page))
	require.Equal(t, 1, page.TotalCount)
	require.Len(t, page.VulRecords, 1)

	rec := page.VulRecords[0]
	// PrimaryId 在云侧是数字，产品用 flexString 兼容数字/字符串两种形态。
	require.Equal(t, "8218810662", string(rec.PrimaryID))
	require.Equal(t, 1, int(rec.Status))
	require.Equal(t, "asap", rec.Necessity)
	require.Equal(t, "i-2zeb9hx05c68a5tlazqm", rec.InstanceID)
	require.Equal(t, "cn-beijing", rec.RegionID)
	require.Equal(t, "OpenClaw 多个安全漏洞合集【版本对比】", rec.Title())
	require.Equal(t, 8.8, rec.CVSSScore())
	// 8.8 → high（>=7 且 <9）。
	require.Equal(t, "high", rec.Severity())

	// 时间戳必须是毫秒解析：2026-06-14T05:10:40Z。
	require.Equal(t, time.Date(2026, 6, 14, 5, 10, 40, 0, time.UTC), rec.FirstSeen())
	require.Equal(t, time.Date(2026, 6, 14, 5, 10, 40, 0, time.UTC), rec.LastSeen())
	require.True(t, rec.FirstSeen().Before(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)),
		"发现时间不得落在公元 58655 年这类荒谬区间")
}

// TestRealAccountVulStatisticsParsesCorrectly 用真实账号的统计响应确认
// 云侧计数字段映射正确——交叉校验闸门完全依赖这个映射。
func TestRealAccountVulStatisticsParsesCorrectly(t *testing.T) {
	const raw = `{
	  "AppCnt": 0, "VulDealedTotalNum": 0, "VulLaterSum": 0,
	  "ImageVulTotalSum": 0, "EmgNum": 1, "AppNum": 0, "ImageVulAsapSum": 0,
	  "VulAsapSum": 1, "VulNntfSum": 0, "CveNum": 0, "ScaNum": 0, "CmsNum": 0,
	  "CmsDealedTotalNum": 0, "SysNum": 0, "ImageVulLaterSum": 0, "ImageVulNntfSum": 0
	}`

	var stats aliyun.SecurityCenterVulStatistics
	require.NoError(t, json.Unmarshal([]byte(raw), &stats))
	require.Equal(t, 1, stats.EmgNum)
	require.Equal(t, 1, stats.VulAsapSum)
	require.Equal(t, 0, stats.CveNum)
	require.Equal(t, 0, stats.SysNum)
	require.Equal(t, 0, stats.CmsNum)
	require.Equal(t, 1, stats.TotalUnfixed())
}

// TestEvidenceFileHasNoCredentials 证据文件不得含凭据。
func TestEvidenceFileHasNoCredentials(t *testing.T) {
	path := os.Getenv("ALISEC_EVIDENCE_PATH")
	if path == "" {
		t.Skip("未设置证据文件路径，跳过")
	}
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	body := string(data)
	require.NotContains(t, body, "LTAI", "证据文件不得含 AccessKey")
	require.NotContains(t, body, "AccessKeySecret")
	require.NotContains(t, body, "Signature=")
}

// TestProbeSecurityCenterRequiresAllowlistedAction 确认探测入口在发请求前
// 就把非只读动作挡掉——用不存在的凭据调用，若守卫失效会得到签名/鉴权错误
// 而不是白名单错误。
func TestProbeSecurityCenterRequiresAllowlistedAction(t *testing.T) {
	_, err := aliyun.ProbeSecurityCenter(context.Background(), "ak", "sk", "", "OpenVulScan", nil)
	require.Error(t, err)
	require.IsType(t, &aliyun.ErrNotReadOnlyAction{}, err)
}
