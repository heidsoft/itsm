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

	t.Run("iam_role valid", func(t *testing.T) {
		cred, err := ResolveAWSCredential(ctx, `{"type":"iam_role"}`)
		require.NoError(t, err)
		assert.Equal(t, "aws", cred.Provider)
		assert.Empty(t, cred.AccessKeyID)
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
