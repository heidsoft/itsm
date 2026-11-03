package aliyun

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// emgRecordObjectShape 是 2026-11-03 真实账号（cn-beijing）DescribeVulList
// 返回的 emg 记录原样：ExtendContentJson 是 **JSON 对象** 而不是字符串。
//
// 这条夹具是必须的：按字符串解码会让 emg 类型整轮同步失败
// （json: cannot unmarshal object into Go struct field ... of type string），
// 该类型的漏洞一条都进不来；而 cve/sys/cms 类型返回的是字符串形态，
// 两种形态必须同时成立。
const emgRecordObjectShape = `{
  "PrimaryId": 8218810662,
  "Status": 1,
  "Name": "emg:SCA:AVD-2026-1860246",
  "AliasName": "OpenClaw 多个安全漏洞合集【版本对比】",
  "Type": "emg",
  "Necessity": "asap",
  "Level": "high",
  "CanFix": "yes",
  "RealRisk": true,
  "InstanceId": "i-example00000000",
  "InstanceName": "itsm-verify",
  "RegionId": "cn-beijing",
  "Uuid": "00000000-0000-0000-0000-000000000000",
  "FirstTs": 1781413840000,
  "LastTs": 1781413840000,
  "ModifyTs": 1781413840000,
  "ExtendContentJson": {
    "Necessity": {},
    "cveList": [],
    "PreCheck": {},
    "AgentScanVul": [],
    "RpmEntityList": [
      {"Path": "/usr/bin/example", "Version": "1.0.0", "Name": "example"}
    ]
  }
}`

// cveRecordStringShape 是同一接口在 cve/sys/cms 类型下的形态：
// ExtendContentJson 是 **JSON 字符串**。
const cveRecordStringShape = `{
  "PrimaryId": "129833",
  "Status": "1",
  "Name": "CVE-2026-0001",
  "AliasName": "示例漏洞",
  "Type": "cve",
  "Necessity": "asap",
  "InstanceId": "i-example00000000",
  "RegionId": "cn-beijing",
  "FirstTs": 1781413840000,
  "LastTs": 1781413840000,
  "ExtendContentJson": "{\"cvss_score\":\"9.8\"}"
}`

// TestSecurityVulnerabilityDecodesObjectShapedExtendContent 锁住对象形态。
func TestSecurityVulnerabilityDecodesObjectShapedExtendContent(t *testing.T) {
	var rec SecurityVulnerability
	require.NoError(t, json.Unmarshal([]byte(emgRecordObjectShape), &rec),
		"emg 类型的对象形态 ExtendContentJson 必须能解码")

	require.Equal(t, "8218810662", string(rec.PrimaryID))
	require.Equal(t, 1, int(rec.Status))
	require.Equal(t, "emg", rec.Type)
	require.Equal(t, "asap", rec.Necessity)
	require.Equal(t, "high", rec.Level)
	require.Equal(t, "i-example00000000", rec.InstanceID)

	// 归一后是 JSON 文本，内容与云侧一致。
	require.NotEmpty(t, rec.ExtendContentJSON)
	require.Contains(t, rec.ExtendContentJSON, "RpmEntityList")

	// emg 没有 cvss_score，严重度必须来自云侧的 Level 字段，
	// 而不是退化成 unknown。
	require.Equal(t, "high", rec.Severity())
	require.Equal(t, float64(0), rec.CVSSScore())

	// 时间戳按毫秒解析：1781413840000 ms = 2026-06-14T05:10:40Z
	require.Equal(t, time.Date(2026, 6, 14, 5, 10, 40, 0, time.UTC), rec.FirstSeen())
	require.Equal(t, time.Date(2026, 6, 14, 5, 10, 40, 0, time.UTC), rec.LastSeen())
}

// TestSecurityVulnerabilityDecodesStringShapedExtendContent 锁住字符串形态
// （原有行为不得被对象形态的兼容改坏）。
func TestSecurityVulnerabilityDecodesStringShapedExtendContent(t *testing.T) {
	var rec SecurityVulnerability
	require.NoError(t, json.Unmarshal([]byte(cveRecordStringShape), &rec))

	require.Equal(t, "129833", string(rec.PrimaryID))
	require.Equal(t, 1, int(rec.Status))
	require.Equal(t, "示例漏洞", rec.Title())
	require.Equal(t, 9.8, rec.CVSSScore())
	require.Equal(t, "critical", rec.Severity())
	require.Contains(t, rec.ExtendContentJSON, "cvss_score")
}

// TestSeverityPrefersCloudLevel 锁住严重度优先级：云侧 Level 优先于 CVSS 推导。
// 两者一致时结果相同；只有 Level 时用它；只有 CVSS 时用它；都没有才是 unknown。
func TestSeverityPrefersCloudLevel(t *testing.T) {
	withLevel := SecurityVulnerability{Level: "medium"}
	require.Equal(t, "medium", withLevel.Severity())

	critical := SecurityVulnerability{Level: "critical"}
	require.Equal(t, "critical", critical.Severity())

	unknownWord := SecurityVulnerability{Level: "not-a-level"}
	require.Equal(t, "unknown", unknownWord.Severity(), "未知 Level 词表应回退到 CVSS/未知")

	// Level 与 CVSS 同时存在时以 Level 为准（云侧等级是权威口径）。
	both := SecurityVulnerability{Level: "low", ExtendContentJSON: `{"cvss_score":"9.8"}`}
	require.Equal(t, "low", both.Severity())

	// 无 Level 时仍走 CVSS 推导。
	onlyCVSS := SecurityVulnerability{ExtendContentJSON: `{"cvss_score":"5.0"}`}
	require.Equal(t, "medium", onlyCVSS.Severity())
}

// TestSecurityVulnerabilityRoundTripsThroughMarshal 归一后的记录要能被再次
// 编回 JSON（漏洞原文会存进 security_findings.raw_payload）。
func TestSecurityVulnerabilityRoundTripsThroughMarshal(t *testing.T) {
	var rec SecurityVulnerability
	require.NoError(t, json.Unmarshal([]byte(emgRecordObjectShape), &rec))

	encoded, err := json.Marshal(rec)
	require.NoError(t, err)
	require.Contains(t, string(encoded), "RpmEntityList")

	var again SecurityVulnerability
	require.NoError(t, json.Unmarshal(encoded, &again))
	require.Equal(t, rec.PrimaryID, again.PrimaryID)
	require.Equal(t, rec.Severity(), again.Severity())
	require.Equal(t, rec.FirstSeen(), again.FirstSeen())
}

// TestFlexJSONObjectToleratesMalformedStringContent 锁住「单条脏数据不拖垮整轮」
// 的取向：ExtendContentJson 是字符串但内容不是合法 JSON 时，**不报错**。
//
// 报错会让这一类型的整轮同步失败（一条漏洞都进不来），而容忍的代价只是丢掉
// 这一条的扩展内容——主字段（Name/Status/InstanceId/Level/时间戳）都还在，
// 漏洞本身仍能入库，严重度退回 Level 或 unknown。宁可少一个评分，不可整类丢光。
func TestFlexJSONObjectToleratesMalformedStringContent(t *testing.T) {
	var rec SecurityVulnerability
	err := json.Unmarshal([]byte(`{"ExtendContentJson":"{not json"}`), &rec)
	require.NoError(t, err, "单条扩展内容脏数据不得让整轮同步失败")
	require.Equal(t, float64(0), rec.CVSSScore(), "内容非法时评分取 0，不猜测")
	require.Equal(t, "unknown", rec.Severity(), "既无评分也无 Level 时是 unknown")

	// 同一批里的正常记录仍要能正常解码——容错不能变成「整批都丢」。
	var good SecurityVulnerability
	require.NoError(t, json.Unmarshal([]byte(cveRecordStringShape), &good))
	require.Equal(t, "critical", good.Severity())
}
