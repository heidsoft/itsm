// Package invocationid 提供 AI 工具 invocation 主体身份的版本化存储信封。
//
// 背景：ToolInvocation 此前没有 principal 列，执行链用两个**间接信号**推断主体：
//   - RevalidateApprovedTool：policy_version 非空 → agent，否则 human
//   - ReviewInvocation：approval_state 为 pending_agent* → agent，否则 human
//
// 两个推断互相分叉，且都把 service 主体错认成 agent（service 走 Gate 2.5，
// policy_version 非空；userID=0 在 agent 分支被 evaluateToolCall 拒绝）。
// 这里在已有的 arguments 文本列保存服务端生成的版本化信封，显式记录主体，
// 不新增 ent 列、不把身份塞进 policy_version / role_snapshot / approval_state。
//
// 契约：
//   - 信封只能由服务端写入；客户端/模型提交的 _itsmInvocation 在编码时被覆盖。
//   - 解码失败一律 fail-closed（ErrInvalid），不静默降级为 human。
//   - 无信封的历史记录走唯一兼容推断：userID=0 无法安全还原 → 拒绝重放。
//   - 返回给工具的参数永远是裸 args，信封不外泄。
package invocationid

import (
	"encoding/json"
	"errors"
	"fmt"

	"itsm-backend/internal/authz"
)

var ErrInvalid = errors.New("invocation identity envelope is invalid")

const (
	envelopeKey = "_itsmInvocation"
	version1    = 1

	// SourceStored / SourceInferred 区分「明确存了」与「历史推断」，供审计展示。
	SourceStored   = "stored"
	SourceInferred = "inferred"
)

type envelope struct {
	Version   int    `json:"version"`
	Principal string `json:"principal"`
}

// Decoded 是一次解码的结果。Args 已剔除信封，可直接传给工具。
type Decoded struct {
	Args      map[string]interface{}
	Principal string
	Source    string
}

var validPrincipals = map[string]bool{
	authz.PrincipalHuman:   true,
	authz.PrincipalAgent:   true,
	authz.PrincipalService: true,
}

// Encode 把工具参数与服务端判定的主体封装成 arguments 文本。
// 客户端参数里若带 _itsmInvocation 会被无条件覆盖，杜绝身份伪造。
func Encode(args map[string]interface{}, principal string) (string, error) {
	if !validPrincipals[principal] {
		return "", fmt.Errorf("%w: unknown principal %q", ErrInvalid, principal)
	}
	if args == nil {
		args = map[string]interface{}{}
	}
	payload := make(map[string]interface{}, len(args)+1)
	for k, v := range args {
		if k == envelopeKey {
			continue // 服务端权威覆盖
		}
		payload[k] = v
	}
	payload[envelopeKey] = envelope{Version: version1, Principal: principal}
	out, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode invocation arguments: %w", err)
	}
	return string(out), nil
}

// Decode 解出裸参数与主体。优先读服务端信封；无信封时走唯一兼容推断。
// 任何无法唯一、安全还原的情况都 fail-closed。
func Decode(raw, policyVersion string, userID int) (*Decoded, error) {
	args := map[string]interface{}{}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &args); err != nil {
			return nil, fmt.Errorf("%w: arguments unreadable", ErrInvalid)
		}
	}
	if args == nil {
		args = map[string]interface{}{}
	}

	envRaw, ok := args[envelopeKey]
	if !ok {
		principal, err := inferLegacy(policyVersion, userID)
		if err != nil {
			return nil, err
		}
		return &Decoded{Args: args, Principal: principal, Source: SourceInferred}, nil
	}
	delete(args, envelopeKey)

	envBytes, err := json.Marshal(envRaw)
	if err != nil {
		return nil, fmt.Errorf("%w: envelope not serializable", ErrInvalid)
	}
	var env envelope
	if err := json.Unmarshal(envBytes, &env); err != nil {
		return nil, fmt.Errorf("%w: envelope unreadable", ErrInvalid)
	}
	if env.Version != version1 || !validPrincipals[env.Principal] {
		return nil, fmt.Errorf("%w: version=%d principal=%q", ErrInvalid, env.Version, env.Principal)
	}
	// service 主体必须是 userID=0；human/agent 必须有正数 userID。
	if env.Principal == authz.PrincipalService {
		if userID != 0 {
			return nil, fmt.Errorf("%w: service principal requires user_id=0", ErrInvalid)
		}
	} else if userID <= 0 {
		return nil, fmt.Errorf("%w: %s principal requires positive user_id", ErrInvalid, env.Principal)
	}
	return &Decoded{Args: args, Principal: env.Principal, Source: SourceStored}, nil
}

// inferLegacy 是存量无信封记录的唯一兼容推断。three-signal 无法交叉时拒绝：
// userID=0 历史上只可能是 service，但旧格式无法证明其载体角色，fail-closed。
func inferLegacy(policyVersion string, userID int) (string, error) {
	if userID <= 0 {
		return "", fmt.Errorf("%w: legacy record with user_id=0 cannot be safely replayed", ErrInvalid)
	}
	if policyVersion != "" {
		return authz.PrincipalAgent, nil
	}
	return authz.PrincipalHuman, nil
}
