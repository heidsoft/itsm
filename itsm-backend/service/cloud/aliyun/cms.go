package aliyun

import (
	"context"
	"strconv"
	"time"
)

const (
	cloudMonitorEndpoint = "metrics.cn-hangzhou.aliyuncs.com"
	cloudMonitorVersion  = "2019-01-01"
)

// CloudMonitorAlertLog 是 CloudMonitor 报警历史中的一条记录。报警历史只包含
// 已经触发的报警，没有 OK 状态，因此同步侧把它当作一次 firing 事件。
type CloudMonitorAlertLog struct {
	LogID        flexString `json:"LogId"`
	AlertTime    string     `json:"AlertTime"`
	Level        string     `json:"Level"`
	RuleID       string     `json:"RuleId"`
	RuleName     string     `json:"RuleName"`
	Namespace    string     `json:"Namespace"`
	Product      string     `json:"Product"`
	MetricName   string     `json:"MetricName"`
	InstanceID   string     `json:"InstanceId"`
	InstanceName string     `json:"InstanceName"`
	Dimensions   string     `json:"Dimensions"`
	Message      string     `json:"Message"`
	GroupID      string     `json:"GroupId"`
}

type alertLogPage struct {
	// 线上 DescribeAlertLogList 成功时返回数字 200，而错误分支返回字符串
	//（如 "Throttling"），因此按 flexString 解码，不能写死 string。
	Code         flexString             `json:"Code"`
	Message      string                 `json:"Message"`
	Success      bool                   `json:"Success"`
	Total        int                    `json:"Total"`
	PageNumber   int                    `json:"PageNumber"`
	PageSize     int                    `json:"PageSize"`
	AlertLogList []CloudMonitorAlertLog `json:"AlertLogList"`
}

// CloudMonitorClient 只读封装 CloudMonitor 报警历史接口。
type CloudMonitorClient struct {
	caller rpcCaller
}

func NewCloudMonitorClient(accessKeyID, accessKeySecret, securityToken string) *CloudMonitorClient {
	return &CloudMonitorClient{
		caller: newRPCClient(cloudMonitorEndpoint, cloudMonitorVersion, accessKeyID, accessKeySecret, securityToken),
	}
}

// DescribeAlertLogs 查询时间窗内的报警历史。时间区间最大 15 天，单页上限由服务端限制。
func (c *CloudMonitorClient) DescribeAlertLogs(ctx context.Context, start, end time.Time, page, pageSize int) ([]CloudMonitorAlertLog, int, error) {
	params := map[string]string{
		"StartTime":  strconv.FormatInt(start.UnixMilli(), 10),
		"EndTime":    strconv.FormatInt(end.UnixMilli(), 10),
		"PageNumber": strconv.Itoa(page),
		"PageSize":   strconv.Itoa(pageSize),
	}

	var out alertLogPage
	if err := c.caller.Call(ctx, "DescribeAlertLogList", params, &out); err != nil {
		return nil, 0, err
	}
	if !out.Success && out.Code != "" && out.Code != "200" {
		return nil, 0, &RPCError{StatusCode: 200, Code: string(out.Code), Message: out.Message}
	}
	return out.AlertLogList, out.Total, nil
}
