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
// 用云侧的 Level 字段或 ExtendContentJson 的 CVSS 评分推导。
type SecurityVulnerability struct {
	PrimaryID    flexString `json:"PrimaryId"`
	Name         string     `json:"Name"`
	AliasName    string     `json:"AliasName"`
	Type         string     `json:"Type"`
	Status       flexInt    `json:"Status"`
	Necessity    string     `json:"Necessity"`
	InstanceID   string     `json:"InstanceId"`
	InstanceName string     `json:"InstanceName"`
	UUID         string     `json:"Uuid"`
	InternetIP   string     `json:"InternetIp"`
	IntranetIP   string     `json:"IntranetIp"`
	RegionID     string     `json:"RegionId"`
	Related      string     `json:"Related"`
	FirstTs      flexInt64  `json:"FirstTs"`
	LastTs       flexInt64  `json:"LastTs"`
	ModifyTs     flexInt64  `json:"ModifyTs"`
	// Level 是云侧给出的漏洞等级（high/medium/low）。emg（应急/SCA）类型
	// 没有 cvss_score，只有 Level——2026-11-03 真实账号实测 Level="high"。
	Level string `json:"Level"`
	// ExtendContentJson 在不同漏洞类型下形态不同（cve/sys/cms 是 JSON 字符串，
	// emg 是 JSON 对象）。解码时统一归一成 JSON 字符串存进来，避免 emg 类型
	// 因「object 不能解进 string」而让整轮同步失败。
	ExtendContentJSON string `json:"ExtendContentJson"`
}

// UnmarshalJSON 解码一条漏洞记录，并把 ExtendContentJson 的两种形态
// （JSON 字符串 / JSON 对象）统一成 JSON 字符串。
//
// 2026-11-03 prod 实测：emg（应急/SCA）类型返回的是 JSON 对象
// {"Necessity":{},"cveList":[],"RpmEntityList":[...]}，按字符串解码会报
// "json: cannot unmarshal object into Go struct field ... of type string"，
// 该类型的漏洞一条都进不来。解码归一后两种形态都能正常落库。
func (v *SecurityVulnerability) UnmarshalJSON(b []byte) error {
	type alias SecurityVulnerability
	aux := struct {
		ExtendContentJSON flexJSONObject `json:"ExtendContentJson"`
		*alias
	}{alias: (*alias)(v)}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	v.ExtendContentJSON = aux.ExtendContentJSON.String()
	return nil
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

// Severity 返回漏洞严重度。
//
// 优先用云侧给的 Level——emg（应急/SCA）类型没有 cvss_score，只有 Level；
// 没有 Level 时才回退到 CVSS 评分推导（cve/sys/cms 常见）。
// 两者都没有时返回 unknown，不猜测。
func (v SecurityVulnerability) Severity() string {
	if level := normalizeSASLevel(v.Level); level != "" {
		return level
	}
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

// normalizeSASLevel 把云安全中心的 Level 词表映射成 ITSM 严重度；
// 未知取值返回空串（由调用方回退到 CVSS 推导）。
func normalizeSASLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium", "mid":
		return "medium"
	case "low":
		return "low"
	default:
		return ""
	}
}

func (v SecurityVulnerability) cvssScore() float64 {
	if v.ExtendContentJSON == "" {
		return 0
	}
	var ext struct {
		CVSSScore flexFloat `json:"cvss_score"`
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
// tsToTime 把云安全中心的时间戳转成 time.Time。
//
// 云安全中心的 FirstTs / LastTs / ModifyTs / RepairTs 官方文档明确「单位为
// 毫秒」（DescribeVulList 响应示例 1620404763000）。此前这里按秒解析，真实
// 数据 1781413840000 会被算成公元 58655 年——漏洞的「首次发现」「最后发现」
// 落在几万年之后，列表排序、水位线比较与"多久没复现"的判断全部失真。
//
// 边界兜底：SAS 不同接口/历史数据可能混用秒级时间戳，用 1e11 分界——
// 毫秒时间戳大于 1e11 才落在 1973 年之后，秒时间戳小于 1e11 才落在
// 5138 年之前，两侧不可能重叠，因此这个判断不会误伤任何真实值。
func tsToTime(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	if value < timestampMillisThreshold {
		return time.Unix(value, 0).UTC()
	}
	return time.UnixMilli(value).UTC()
}

// timestampMillisThreshold 是毫秒/秒时间戳的分界值。
const timestampMillisThreshold = int64(1e11)

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

// SecurityCenterVulStatistics 是云安全中心的漏洞计数（各类型未修复数）。
//
// 这是云侧的权威计数，与「本轮拉到了几条列表」是两件事：前者回答「云上到底
// 有多少漏洞」，后者回答「我们这次同步到了多少」。同步侧用它做交叉校验，
// 避免「漏同步」被「本轮没出现即已修复」的收口误判成全部已修复。
type SecurityCenterVulStatistics struct {
	CveNum int `json:"CveNum"`
	SysNum int `json:"SysNum"`
	CmsNum int `json:"CmsNum"`
	ScaNum int `json:"ScaNum"`
	AppNum int `json:"AppNum"`
	EmgNum int `json:"EmgNum"`
	// VulAsapSum / VulLaterSum / VulNntfSum 按修复必要性分级统计未修复漏洞。
	VulAsapSum  int `json:"VulAsapSum"`
	VulLaterSum int `json:"VulLaterSum"`
	VulNntfSum  int `json:"VulNntfSum"`
	// VulDealedTotalNum 是已修复（已处理）的漏洞总数。
	VulDealedTotalNum int `json:"VulDealedTotalNum"`
}

// CountFor 返回指定漏洞类型的云侧计数；未知类型返回 0。
func (s *SecurityCenterVulStatistics) CountFor(vulnType string) int {
	if s == nil {
		return 0
	}
	switch vulnType {
	case "cve":
		return s.CveNum
	case "sys":
		return s.SysNum
	case "cms":
		return s.CmsNum
	case "sca":
		return s.ScaNum
	case "app":
		return s.AppNum
	case "emg":
		return s.EmgNum
	default:
		return 0
	}
}

// TotalUnfixed 是按修复必要性分级的未修复漏洞总数。
func (s *SecurityCenterVulStatistics) TotalUnfixed() int {
	if s == nil {
		return 0
	}
	return s.VulAsapSum + s.VulLaterSum + s.VulNntfSum
}

// DescribeVulNumStatistics 查询云安全中心的漏洞计数。
func (c *SecurityCenterClient) DescribeVulNumStatistics(ctx context.Context) (*SecurityCenterVulStatistics, error) {
	var out SecurityCenterVulStatistics
	if err := c.caller.Call(ctx, "DescribeVulNumStatistics", map[string]string{"Lang": "zh"}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
