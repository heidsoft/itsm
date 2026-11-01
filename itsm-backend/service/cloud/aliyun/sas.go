package aliyun

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

const (
	securityCenterEndpoint = "tds.aliyuncs.com"
	securityCenterVersion  = "2018-12-03"
	securityCenterRegion   = "cn-hangzhou"
)

// 云安全中心漏洞状态：1 未修复，2 修复中，3 已修复，4 已忽略（其余状态不作为待处理项）。
const (
	VulStatusUnfixed = 1
	VulStatusFixing  = 2
	VulStatusFixed   = 3
	VulStatusIgnored = 4
)

// SecurityVulnerability 是云安全中心一条漏洞记录。Severity 未在列表接口中直接给出，
// 用 ExtendContentJson 的 CVSS 评分推导。
type SecurityVulnerability struct {
	PrimaryID         flexString `json:"PrimaryId"`
	Name              string     `json:"Name"`
	AliasName         string     `json:"AliasName"`
	Type              string     `json:"Type"`
	Status            flexInt    `json:"Status"`
	Necessity         string     `json:"Necessity"`
	InstanceID        string     `json:"InstanceId"`
	InstanceName      string     `json:"InstanceName"`
	UUID              string     `json:"Uuid"`
	InternetIP        string     `json:"InternetIp"`
	IntranetIP        string     `json:"IntranetIp"`
	RegionID          string     `json:"RegionId"`
	Related           string     `json:"Related"`
	FirstTs           flexInt64  `json:"FirstTs"`
	LastTs            flexInt64  `json:"LastTs"`
	ModifyTs          flexInt64  `json:"ModifyTs"`
	ExtendContentJSON string     `json:"ExtendContentJson"`
}

type flexInt64 int64

func (f *flexInt64) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*f = flexInt64(v)
	return nil
}

// Severity 把 CVSS 评分映射为 ITSM 严重度；无评分时返回 unknown，不猜测。
func (v SecurityVulnerability) Severity() string {
	score := v.cvssScore()
	switch {
	case score == 0:
		return "unknown"
	case score >= 9:
		return "critical"
	case score >= 7:
		return "high"
	case score >= 4:
		return "medium"
	default:
		return "low"
	}
}

func (v SecurityVulnerability) cvssScore() float64 {
	var ext struct {
		CVSSScore flexFloat `json:"cvss_score"`
	}
	if v.ExtendContentJSON == "" {
		return 0
	}
	if err := json.Unmarshal([]byte(v.ExtendContentJSON), &ext); err != nil {
		return 0
	}
	return float64(ext.CVSSScore)
}

// CVSSScore 返回漏洞的 CVSS 评分；0 表示上游未给出评分。
func (v SecurityVulnerability) CVSSScore() float64 { return v.cvssScore() }

// Title 优先使用漏洞别名，其次用正式名称。
func (v SecurityVulnerability) Title() string {
	if strings.TrimSpace(v.AliasName) != "" {
		return v.AliasName
	}
	return v.Name
}

func (v SecurityVulnerability) FirstSeen() time.Time { return tsToTime(int64(v.FirstTs)) }
func (v SecurityVulnerability) LastSeen() time.Time  { return tsToTime(int64(v.LastTs)) }

// FirstTs/LastTs 是秒级时间戳；0 表示未知。
func tsToTime(sec int64) time.Time {
	if sec <= 0 {
		return time.Time{}
	}
	return time.Unix(sec, 0).UTC()
}

type vulListPage struct {
	TotalCount  int                     `json:"TotalCount"`
	CurrentPage int                     `json:"CurrentPage"`
	PageSize    int                     `json:"PageSize"`
	VulRecords  []SecurityVulnerability `json:"VulRecords"`
}

// SecurityCenterClient 只读封装云安全中心漏洞列表接口。
type SecurityCenterClient struct {
	caller rpcCaller
}

func NewSecurityCenterClient(accessKeyID, accessKeySecret, securityToken string) *SecurityCenterClient {
	return &SecurityCenterClient{
		caller: newRPCClient(securityCenterEndpoint, securityCenterVersion, accessKeyID, accessKeySecret, securityToken),
	}
}

// DescribeVulnerabilities 查询指定类型与状态的漏洞。vulType 取值 cve/sys/cms/app/emg。
func (c *SecurityCenterClient) DescribeVulnerabilities(ctx context.Context, vulType string, statusList []int, page, pageSize int) ([]SecurityVulnerability, int, error) {
	params := map[string]string{
		"Type":        vulType,
		"CurrentPage": strconv.Itoa(page),
		"PageSize":    strconv.Itoa(pageSize),
		"RegionId":    securityCenterRegion,
	}
	if len(statusList) > 0 {
		values := make([]string, 0, len(statusList))
		for _, s := range statusList {
			values = append(values, strconv.Itoa(s))
		}
		params["StatusList"] = strings.Join(values, ",")
	}

	var out vulListPage
	if err := c.caller.Call(ctx, "DescribeVulList", params, &out); err != nil {
		return nil, 0, err
	}
	return out.VulRecords, out.TotalCount, nil
}
