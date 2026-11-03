package aliyun

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeCaller struct {
	action  string
	params  map[string]string
	respond func(out any) error
}

func (f *fakeCaller) Call(_ context.Context, action string, params map[string]string, out any) error {
	f.action = action
	f.params = params
	if f.respond == nil {
		return nil
	}
	return f.respond(out)
}

func decodeInto(payload string) func(out any) error {
	return func(out any) error { return json.Unmarshal([]byte(payload), out) }
}

func TestCloudMonitorDescribeAlertLogsSendsWindowAndParsesLogID(t *testing.T) {
	caller := &fakeCaller{respond: decodeInto(`{
		"Code": "200",
		"Success": true,
		"Total": 1,
		"AlertLogList": [{
			"LogId": 1770000000123,
			"AlertTime": "2026-10-31 10:00:00",
			"Level": "CRITICAL",
			"RuleId": "rule-1",
			"RuleName": "ECS CPU 高",
			"Namespace": "acs_ecs_dashboard",
			"InstanceId": "i-abc",
			"InstanceName": "web-1",
			"Dimensions": "[{\"instanceId\":\"i-abc\"}]",
			"Message": "cpu_total > 90"
		}]
	}`)}
	client := &CloudMonitorClient{caller: caller}

	start := time.UnixMilli(1770000000000)
	end := start.Add(5 * time.Minute)
	logs, total, err := client.DescribeAlertLogs(context.Background(), start, end, 2, 100)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, logs, 1)
	require.Equal(t, "1770000000123", string(logs[0].LogID))
	require.Equal(t, "rule-1", logs[0].RuleID)
	require.Equal(t, "DescribeAlertLogList", caller.action)
	require.Equal(t, "1770000000000", caller.params["StartTime"])
	require.Equal(t, "1770000300000", caller.params["EndTime"])
	require.Equal(t, "2", caller.params["PageNumber"])
	require.Equal(t, "100", caller.params["PageSize"])
}

func TestCloudMonitorDescribeAlertLogsSurfacesBusinessError(t *testing.T) {
	client := &CloudMonitorClient{caller: &fakeCaller{respond: decodeInto(`{
		"Code": "Throttling",
		"Message": "request was denied due to request throttling",
		"Success": false
	}`)}}

	_, _, err := client.DescribeAlertLogs(context.Background(), time.Now().Add(-time.Hour), time.Now(), 1, 10)

	var rpcErr *RPCError
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, "Throttling", rpcErr.Code)
}

func TestCloudMonitorDescribeAlertLogsAcceptsNumericCode(t *testing.T) {
	// 线上 DescribeAlertLogList 成功响应实际返回数字 Code，此前按 string 解码
	// 导致整个告警同步在解码阶段失败。
	client := &CloudMonitorClient{caller: &fakeCaller{respond: decodeInto(`{
		"Code": 200,
		"Success": true,
		"Total": 1,
		"AlertLogList": [{
			"LogId": 1770000000123,
			"AlertTime": "2026-10-31 10:00:00",
			"Level": "CRITICAL",
			"RuleId": "rule-1"
		}]
	}`)}}

	logs, total, err := client.DescribeAlertLogs(context.Background(), time.Now().Add(-time.Hour), time.Now(), 1, 10)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, logs, 1)
	require.Equal(t, "rule-1", logs[0].RuleID)
}

func TestCloudMonitorDescribeAlertLogsSurfacesNumericBusinessError(t *testing.T) {
	client := &CloudMonitorClient{caller: &fakeCaller{respond: decodeInto(`{
		"Code": 503,
		"Message": "service unavailable",
		"Success": false
	}`)}}

	_, _, err := client.DescribeAlertLogs(context.Background(), time.Now().Add(-time.Hour), time.Now(), 1, 10)

	var rpcErr *RPCError
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, "503", rpcErr.Code)
	require.Equal(t, "service unavailable", rpcErr.Message)
}

func TestParseRPCErrorAcceptsNumericCode(t *testing.T) {
	err := parseRPCError(500, []byte(`{"Code": 500,"Message":"internal error","RequestId":"req-1"}`))

	var rpcErr *RPCError
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, "500", rpcErr.Code)
	require.Equal(t, "internal error", rpcErr.Message)
	require.Equal(t, "req-1", rpcErr.RequestID)
}

func TestSecurityCenterDescribeVulnerabilitiesParsesRecords(t *testing.T) {
	caller := &fakeCaller{respond: decodeInto(`{
		"TotalCount": 1,
		"CurrentPage": 1,
		"PageSize": 100,
		"VulRecords": [{
			"PrimaryId": 129833,
			"Name": "CVE-2024-1234",
			"AliasName": "OpenSSL 缓冲区溢出",
			"Type": "cve",
			"Status": "1",
			"Necessity": "asap",
			"InstanceId": "i-abc",
			"InstanceName": "web-1",
			"RegionId": "cn-beijing",
			"FirstTs": "1760000000",
			"LastTs": 1760100000,
			"ExtendContentJson": "{\"cvss_score\":\"9.8\"}"
		}]
	}`)}
	client := &SecurityCenterClient{caller: caller}

	records, total, err := client.DescribeVulnerabilities(context.Background(), "cve", []int{VulStatusUnfixed, VulStatusFixing}, 1, 100)
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, records, 1)
	require.Equal(t, "129833", string(records[0].PrimaryID))
	require.Equal(t, 1, int(records[0].Status))
	require.Equal(t, "OpenSSL 缓冲区溢出", records[0].Title())
	require.Equal(t, "critical", records[0].Severity())
	require.Equal(t, time.Unix(1760000000, 0).UTC(), records[0].FirstSeen())
	require.Equal(t, "DescribeVulList", caller.action)
	require.Equal(t, "cve", caller.params["Type"])
	require.Equal(t, "1,2", caller.params["StatusList"])
	require.Equal(t, "cn-hangzhou", caller.params["RegionId"])
}

func TestVulnerabilitySeverityMapping(t *testing.T) {
	cases := []struct {
		extend   string
		expected string
	}{
		{`{"cvss_score":9.8}`, "critical"},
		{`{"cvss_score":"7.5"}`, "high"},
		{`{"cvss_score":"5.0"}`, "medium"},
		{`{"cvss_score":"2.1"}`, "low"},
		{`{}`, "unknown"},
		{``, "unknown"},
	}
	for _, tc := range cases {
		v := SecurityVulnerability{ExtendContentJSON: tc.extend}
		require.Equal(t, tc.expected, v.Severity(), tc.extend)
	}
}
