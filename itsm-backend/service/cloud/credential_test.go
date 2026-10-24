package cloud

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCredentialRef(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(*testing.T, map[string]interface{})
	}{
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   "{not json}",
			wantErr: true,
		},
		{
			name:    "valid JSON",
			input:   `{"type":"ak","access_key_id":"foo"}`,
			wantErr: false,
			check: func(t *testing.T, data map[string]interface{}) {
				assert.Equal(t, "ak", data["type"])
				assert.Equal(t, "foo", data["access_key_id"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := ParseCredentialRef(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, data)
			}
		})
	}
}

func TestResolveAliyunCredential(t *testing.T) {
	ctx := context.Background()

	t.Run("ak valid", func(t *testing.T) {
		cred, err := ResolveAliyunCredential(ctx, `{"type":"ak","access_key_id":"foo","access_key_secret":"bar"}`)
		require.NoError(t, err)
		assert.Equal(t, "aliyun", cred.Provider)
		assert.Equal(t, "foo", cred.AccessKeyID)
		assert.Equal(t, "bar", cred.AccessKeySecret)
		assert.Empty(t, cred.SessionToken)
	})

	t.Run("sts valid", func(t *testing.T) {
		cred, err := ResolveAliyunCredential(ctx, `{"type":"sts","access_key_id":"foo","access_key_secret":"bar","session_token":"tok123"}`)
		require.NoError(t, err)
		assert.Equal(t, "aliyun", cred.Provider)
		assert.Equal(t, "foo", cred.AccessKeyID)
		assert.Equal(t, "bar", cred.AccessKeySecret)
		assert.Equal(t, "tok123", cred.SessionToken)
	})

	t.Run("ram_role valid", func(t *testing.T) {
		cred, err := ResolveAliyunCredential(ctx, `{"type":"ram_role","role_arn":"acs:ram::123:role/ECSAdmin","session_name":"ecs-discovery"}`)
		require.NoError(t, err)
		assert.Equal(t, "aliyun", cred.Provider)
		assert.Equal(t, "acs:ram::123:role/ECSAdmin", cred.RoleARN)
		assert.Equal(t, "ecs-discovery", cred.SessionName)
	})

	t.Run("ak missing access_key_id", func(t *testing.T) {
		_, err := ResolveAliyunCredential(ctx, `{"type":"ak","access_key_secret":"bar"}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access_key_id")
	})

	t.Run("ak missing access_key_secret", func(t *testing.T) {
		_, err := ResolveAliyunCredential(ctx, `{"type":"ak","access_key_id":"foo"}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access_key_secret")
	})

	t.Run("unknown credential type", func(t *testing.T) {
		_, err := ResolveAliyunCredential(ctx, `{"type":"unknown"}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown")
	})

	t.Run("empty credential_ref", func(t *testing.T) {
		_, err := ResolveAliyunCredential(ctx, "")
		require.Error(t, err)
	})

	t.Run("environment reference", func(t *testing.T) {
		t.Setenv("ITSM_ALIYUN_ACCESS_KEY_ID", "env-id")
		t.Setenv("ITSM_ALIYUN_ACCESS_KEY_SECRET", "env-secret")
		t.Setenv("ITSM_ALIYUN_SECURITY_TOKEN", "env-token")

		cred, err := ResolveAliyunCredential(ctx, "env://ITSM_ALIYUN")
		require.NoError(t, err)
		assert.Equal(t, "env-id", cred.AccessKeyID)
		assert.Equal(t, "env-secret", cred.AccessKeySecret)
		assert.Equal(t, "env-token", cred.SessionToken)
	})

	t.Run("rejects arbitrary environment prefix", func(t *testing.T) {
		_, err := ResolveAliyunCredential(ctx, "env://HOME")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported")
	})
}

func TestResolveAWSCredential(t *testing.T) {
	ctx := context.Background()

	t.Run("keys valid", func(t *testing.T) {
		cred, err := ResolveAWSCredential(ctx, `{"type":"keys","access_key_id":"AKIAIOSFODNN7EXAMPLE","secret_access_key":"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"}`)
		require.NoError(t, err)
		assert.Equal(t, "aws", cred.Provider)
		assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", cred.AccessKeyID)
	})

	t.Run("keys missing secret", func(t *testing.T) {
		_, err := ResolveAWSCredential(ctx, `{"type":"keys","access_key_id":"foo"}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret_access_key")
	})

	t.Run("keys missing access key id", func(t *testing.T) {
		// 原先只校验 secret 不校验 id：漏填 access_key_id 也能「解析成功」，
		// 拿到的空 AK 签出来的请求必然 AuthFailure，而错误指向签名而不是配置。
		_, err := ResolveAWSCredential(ctx, `{"type":"keys","secret_access_key":"secret"}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access_key_id")
	})

	t.Run("endpoint override", func(t *testing.T) {
		cred, err := ResolveAWSCredential(ctx, `{"type":"keys","access_key_id":"AKIA1","secret_access_key":"s","endpoint":"https://ec2.us-gov-west-1.amazonaws.com"}`)
		require.NoError(t, err)
		assert.Equal(t, "https://ec2.us-gov-west-1.amazonaws.com", cred.Endpoint)
	})

	t.Run("iam_role fails closed", func(t *testing.T) {
		// 此前这一支返回一份 AK/SK 皆空的凭据并判为成功。在没有任何消费者时
		// 这个空凭据无害；但 AWS 适配器（service/cloud/aws）落地后它会被直接
		// 送进 SigV4 签名，产出必然失败、且错误指向签名算法的请求。
		//
		// 真实的 iam_role 要走实例元数据服务（IMDSv2）取临时凭据，而本产品通常
		// 并不跑在 EC2 上，实现它收益有限。所以显式报错而不是静默给空凭据——
		// fail closed，并把「未实现」说清楚。
		_, err := ResolveAWSCredential(ctx, `{"type":"iam_role"}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "iam_role")
	})
}

func TestResolveTencentCredential(t *testing.T) {
	ctx := context.Background()

	t.Run("ak valid", func(t *testing.T) {
		cred, err := resolveTencentCredential(ctx, `{"type":"ak","secret_id":"AKIDBBBBBBBBBBBBBBBBBB","secret_key":"secretkey123456"}`)
		require.NoError(t, err)
		assert.Equal(t, "tencent", cred.Provider)
		assert.Equal(t, "AKIDBBBBBBBBBBBBBBBBBB", cred.AccessKeyID)
		assert.Equal(t, "secretkey123456", cred.AccessKeySecret)
	})

	t.Run("ak missing secret_key", func(t *testing.T) {
		_, err := resolveTencentCredential(ctx, `{"type":"ak","secret_id":"foo"}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret_key")
	})

	t.Run("unknown type", func(t *testing.T) {
		_, err := resolveTencentCredential(ctx, `{"type":"role"}`)
		require.Error(t, err)
	})

	t.Run("endpoint override", func(t *testing.T) {
		cred, err := resolveTencentCredential(ctx, `{"type":"ak","secret_id":"id","secret_key":"key","endpoint":"https://cvm.internal.example.com"}`)
		require.NoError(t, err)
		assert.Equal(t, "https://cvm.internal.example.com", cred.Endpoint)
	})
}

func TestResolveVolcengineCredential(t *testing.T) {
	ctx := context.Background()

	t.Run("ak valid", func(t *testing.T) {
		cred, err := ResolveVolcengineCredential(ctx, `{"type":"ak","access_key_id":"AKLTTEST","access_key_secret":"secret"}`)
		require.NoError(t, err)
		assert.Equal(t, "volcengine", cred.Provider)
		assert.Equal(t, "AKLTTEST", cred.AccessKeyID)
		assert.Equal(t, "secret", cred.AccessKeySecret)
		assert.Empty(t, cred.SessionToken)
	})

	t.Run("missing access_key_id", func(t *testing.T) {
		_, err := ResolveVolcengineCredential(ctx, `{"type":"ak","access_key_secret":"secret"}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access_key_id")
	})

	t.Run("sts requires session_token", func(t *testing.T) {
		// STS 临时凭据少了 token 就是一份不完整的凭据：签名能算出来，但服务端
		// 一定拒绝。宁可在解析阶段拒绝。
		_, err := ResolveVolcengineCredential(ctx, `{"type":"sts","access_key_id":"a","access_key_secret":"b"}`)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "session_token")

		cred, err := ResolveVolcengineCredential(ctx, `{"type":"sts","access_key_id":"a","access_key_secret":"b","session_token":"t"}`)
		require.NoError(t, err)
		assert.Equal(t, "t", cred.SessionToken)
	})

	t.Run("unknown type", func(t *testing.T) {
		_, err := ResolveVolcengineCredential(ctx, `{"type":"ram_role"}`)
		require.Error(t, err)
	})
}

func TestResolveCredential_Routing(t *testing.T) {
	ctx := context.Background()

	t.Run("aliyun routes correctly", func(t *testing.T) {
		cred, err := ResolveCredential(ctx, "aliyun", `{"type":"ak","access_key_id":"x","access_key_secret":"y"}`)
		require.NoError(t, err)
		assert.Equal(t, "aliyun", cred.Provider)
	})

	t.Run("aws routes correctly", func(t *testing.T) {
		cred, err := ResolveCredential(ctx, "aws", `{"type":"keys","access_key_id":"x","secret_access_key":"y"}`)
		require.NoError(t, err)
		assert.Equal(t, "aws", cred.Provider)
	})

	t.Run("tencent routes correctly", func(t *testing.T) {
		cred, err := ResolveCredential(ctx, "tencent", `{"type":"ak","secret_id":"x","secret_key":"y"}`)
		require.NoError(t, err)
		assert.Equal(t, "tencent", cred.Provider)
	})

	t.Run("aliyun alias alibaba routes to aliyun", func(t *testing.T) {
		cred, err := ResolveCredential(ctx, "alibaba", `{"type":"ak","access_key_id":"x","access_key_secret":"y"}`)
		require.NoError(t, err)
		assert.Equal(t, "aliyun", cred.Provider)
	})

	t.Run("tencent alias qcloud routes to tencent", func(t *testing.T) {
		cred, err := ResolveCredential(ctx, "qcloud", `{"type":"ak","secret_id":"x","secret_key":"y"}`)
		require.NoError(t, err)
		assert.Equal(t, "tencent", cred.Provider)
	})

	t.Run("unsupported provider", func(t *testing.T) {
		_, err := ResolveCredential(ctx, "unknown_cloud", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported")
	})

	t.Run("azure not implemented", func(t *testing.T) {
		_, err := ResolveCredential(ctx, "azure", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
	})
}
