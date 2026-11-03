package aliyun

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestTimestampsAreMilliseconds 锁住云安全中心时间戳的单位。
//
// 官方 DescribeVulList 文档明确 FirstTs/LastTs/ModifyTs/RepairTs「单位为
// 毫秒」（响应示例 1620404763000）。2026-11-03 真实账号实测值 1781413840000
// 也是毫秒。此前按秒解析，真实数据会被算成公元 58655 年——漏洞的「首次
// 发现」「最后发现」落在几万年之后，排序与水位线比较全部失真。
func TestTimestampsAreMilliseconds(t *testing.T) {
	// 真实账号实测值：1781413840000 ms = 2026-06-14T05:10:40Z
	got := tsToTime(1781413840000)
	require.Equal(t, time.Date(2026, 6, 14, 5, 10, 40, 0, time.UTC), got)
	require.True(t, got.After(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		"毫秒时间戳不得被解析到 2026 年之前")
	require.True(t, got.Before(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)),
		"毫秒时间戳不得被解析到 2027 年之后")
}

// TestTimestampsFallBackToSeconds 保留秒级兜底：SAS 不同接口/历史数据可能
// 混用秒级时间戳，1e11 是不会误伤任何真实值的分界。
func TestTimestampsFallBackToSeconds(t *testing.T) {
	require.Equal(t, time.Date(2025, 10, 9, 8, 53, 20, 0, time.UTC),
		tsToTime(1760000000))
	require.Equal(t, time.Date(2026, 6, 14, 5, 10, 40, 0, time.UTC),
		tsToTime(1781413840000))
}

// TestZeroTimestampIsUnknown 0 表示未知，不能变成 1970 年或公元 1 年。
func TestZeroTimestampIsUnknown(t *testing.T) {
	require.True(t, tsToTime(0).IsZero())
	require.True(t, tsToTime(-1).IsZero())
}

// TestVulStatisticsCountFor 锁住云侧计数到漏洞类型的映射。
// 映射错了会让交叉校验闸门误判「云侧有漏洞但我们拉到 0」。
func TestVulStatisticsCountFor(t *testing.T) {
	stats := &SecurityCenterVulStatistics{
		CveNum: 3, SysNum: 1, CmsNum: 0, ScaNum: 2, AppNum: 0, EmgNum: 5,
		VulAsapSum: 4, VulLaterSum: 3, VulNntfSum: 1,
	}
	require.Equal(t, 3, stats.CountFor("cve"))
	require.Equal(t, 1, stats.CountFor("sys"))
	require.Equal(t, 0, stats.CountFor("cms"))
	require.Equal(t, 2, stats.CountFor("sca"))
	require.Equal(t, 0, stats.CountFor("app"))
	require.Equal(t, 5, stats.CountFor("emg"))
	require.Equal(t, 0, stats.CountFor("unknown-type"))
	require.Equal(t, 8, stats.TotalUnfixed())

	var nilStats *SecurityCenterVulStatistics
	require.Equal(t, 0, nilStats.CountFor("cve"))
	require.Equal(t, 0, nilStats.TotalUnfixed())
}
