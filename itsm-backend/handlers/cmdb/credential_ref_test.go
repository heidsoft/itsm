package cmdb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateTenantCredentialRef(t *testing.T) {
	for _, valid := range []string{"", "secret://tenant-42/aliyun-production", "env://ITSM_ALIYUN"} {
		require.NoError(t, validateTenantCredentialRef(valid, 42))
	}
	for _, invalid := range []string{
		`{"access_key_id":"ak","access_key_secret":"secret"}`,
		"env://OTHER_SECRET",
		"aliyun-production",
		"secret://tenant-42/key?token=secret",
		"secret://tenant-99/aliyun-production",
	} {
		require.Error(t, validateTenantCredentialRef(invalid, 42), invalid)
	}
}
