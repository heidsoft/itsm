package cmdb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateTenantCredentialRef(t *testing.T) {
	for _, valid := range []string{"", "secret://tenant-42/aliyun-production"} {
		require.NoError(t, validateTenantCredentialRef(valid))
	}
	for _, invalid := range []string{
		`{"access_key_id":"ak","access_key_secret":"secret"}`,
		"env://ITSM_ALIYUN",
		"aliyun-production",
		"secret://tenant-42/key?token=secret",
	} {
		require.Error(t, validateTenantCredentialRef(invalid), invalid)
	}
}
