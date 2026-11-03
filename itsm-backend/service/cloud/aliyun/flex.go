package aliyun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// flexJSONObject 兼容「JSON 字符串」与「JSON 对象」两种形态的扩展内容字段。
//
// 云安全中心的 ExtendContentJson 在不同漏洞类型下形态不同（2026-11-03 真实
// 账号实测）：
//   - cve/sys/cms：JSON 字符串 —— "{\"cvss_score\":\"8.8\"}"
//   - emg（应急/SCA）：JSON 对象 —— {"cvss_score":..,"Necessity":{},"RpmEntityList":[..]}
//
// 按字符串解码会让 emg 类型的整轮同步失败（json: cannot unmarshal object
// into Go struct field ... of type string），该类型一条漏洞都进不来。
type flexJSONObject struct {
	Raw json.RawMessage
}

// UnmarshalJSON 兼容字符串与对象两种形态；字符串形态解出后再当 JSON 用。
func (f *flexJSONObject) UnmarshalJSON(b []byte) error {
	trimmed := bytes.TrimSpace(b)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		f.Raw = nil
		return nil
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return err
		}
		if strings.TrimSpace(s) == "" {
			f.Raw = nil
			return nil
		}
		f.Raw = json.RawMessage(s)
		return nil
	}
	f.Raw = append(json.RawMessage(nil), trimmed...)
	return nil
}

// MarshalJSON 原样写回原始 JSON，保证存进 security_findings.raw_payload 的
// 内容与云侧一致（能看出它原本是对象还是字符串）。
func (f flexJSONObject) MarshalJSON() ([]byte, error) {
	if len(f.Raw) == 0 {
		return []byte("null"), nil
	}
	return f.Raw, nil
}

// String 返回规范化后的 JSON 文本；无内容时返回空串。
func (f flexJSONObject) String() string {
	if len(f.Raw) == 0 {
		return ""
	}
	return string(f.Raw)
}

// flexString 兼容阿里云把同一个 ID 字段时而输出字符串、时而输出数字。
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		*f = ""
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = flexString(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return fmt.Errorf("flexString: %w", err)
	}
	*f = flexString(n.String())
	return nil
}

// flexFloat 兼容评分字段的字符串/数字两种输出。
type flexFloat float64

func (f *flexFloat) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("flexFloat: %w", err)
	}
	*f = flexFloat(v)
	return nil
}

type flexInt int

func (f *flexInt) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("flexInt: %w", err)
	}
	*f = flexInt(v)
	return nil
}
