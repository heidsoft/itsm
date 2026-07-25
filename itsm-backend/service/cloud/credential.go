package cloud

import (
	"context"
	"encoding/json"
	"fmt"
)

// ResolvedCredential 运行时凭据结构（所有字段非 nil）
type ResolvedCredential struct {
	Provider        string
	AccessKeyID     string
	AccessKeySecret string
	SessionToken    string
	RoleARN         string
	SessionName     string
}

// ParseCredentialRef 解析原始 JSON 字符串
func ParseCredentialRef(credentialRef string) (map[string]interface{}, error) {
	if credentialRef == "" {
		return nil, fmt.Errorf("credential_ref is empty")
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(credentialRef), &data); err != nil {
		return nil, fmt.Errorf("invalid JSON in credential_ref: %w", err)
	}
	return data, nil
}

// ResolveCredential 根据 provider 调用对应解析器
func ResolveCredential(ctx context.Context, provider, credentialRef string) (*ResolvedCredential, error) {
	switch NormalizeProvider(provider) {
	case "aliyun":
		return ResolveAliyunCredential(ctx, credentialRef)
	case "aws":
		return ResolveAWSCredential(ctx, credentialRef)
	case "tencent":
		return resolveTencentCredential(ctx, credentialRef)
	case "azure":
		return nil, fmt.Errorf("azure credential resolver not implemented")
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// ResolveAliyunCredential 解析阿里云凭据
func ResolveAliyunCredential(ctx context.Context, credentialRef string) (*ResolvedCredential, error) {
	data, err := ParseCredentialRef(credentialRef)
	if err != nil {
		return nil, err
	}
	cred := &ResolvedCredential{Provider: "aliyun"}
	switch data["type"] {
	case "ak":
		cred.AccessKeyID, _ = data["access_key_id"].(string)
		cred.AccessKeySecret, _ = data["access_key_secret"].(string)
		if cred.AccessKeyID == "" {
			return nil, fmt.Errorf("access_key_id required for ak credential type")
		}
		if cred.AccessKeySecret == "" {
			return nil, fmt.Errorf("access_key_secret required for ak credential type")
		}
	case "sts":
		cred.AccessKeyID, _ = data["access_key_id"].(string)
		cred.AccessKeySecret, _ = data["access_key_secret"].(string)
		cred.SessionToken, _ = data["session_token"].(string)
		if cred.AccessKeyID == "" {
			return nil, fmt.Errorf("access_key_id required for sts credential type")
		}
		if cred.AccessKeySecret == "" {
			return nil, fmt.Errorf("access_key_secret required for sts credential type")
		}
	case "ram_role":
		cred.RoleARN, _ = data["role_arn"].(string)
		cred.SessionName, _ = data["session_name"].(string)
		if cred.RoleARN == "" {
			return nil, fmt.Errorf("role_arn required for ram_role credential type")
		}
	default:
		return nil, fmt.Errorf("unknown aliyun credential type: %v", data["type"])
	}
	return cred, nil
}

// ResolveAWSCredential 解析 AWS 凭据
func ResolveAWSCredential(ctx context.Context, credentialRef string) (*ResolvedCredential, error) {
	data, err := ParseCredentialRef(credentialRef)
	if err != nil {
		return nil, err
	}
	cred := &ResolvedCredential{Provider: "aws"}
	switch data["type"] {
	case "keys":
		cred.AccessKeyID, _ = data["access_key_id"].(string)
		cred.AccessKeySecret, _ = data["secret_access_key"].(string)
		if cred.AccessKeySecret == "" {
			return nil, fmt.Errorf("secret_access_key required for keys credential type")
		}
	case "iam_role":
		// IAM Role 不需要 AK/SK，由元数据服务自动提供
	default:
		return nil, fmt.Errorf("unknown aws credential type: %v", data["type"])
	}
	return cred, nil
}

// resolveTencentCredential 解析腾讯云凭据
func resolveTencentCredential(ctx context.Context, credentialRef string) (*ResolvedCredential, error) {
	data, err := ParseCredentialRef(credentialRef)
	if err != nil {
		return nil, err
	}
	cred := &ResolvedCredential{Provider: "tencent"}
	switch data["type"] {
	case "ak":
		cred.AccessKeyID, _ = data["secret_id"].(string)
		cred.AccessKeySecret, _ = data["secret_key"].(string)
		if cred.AccessKeyID == "" {
			return nil, fmt.Errorf("secret_id required for tencent ak credential type")
		}
		if cred.AccessKeySecret == "" {
			return nil, fmt.Errorf("secret_key required for tencent ak credential type")
		}
	default:
		return nil, fmt.Errorf("unknown tencent credential type: %v", data["type"])
	}
	return cred, nil
}
