package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent"
)

func TestCloudAccountResponseDoesNotExposeCredential(t *testing.T) {
	response := ToCloudAccountResponse(&ent.CloudAccount{
		ID:            1,
		Provider:      "aliyun",
		AccountID:     "account-1",
		AccountName:   "production",
		CredentialRef: `{"access_key_id":"secret-id","access_key_secret":"secret-value"}`,
		TenantID:      7,
	})

	require.NotNil(t, response)
	assert.True(t, response.HasCredential)

	payload, err := json.Marshal(response)
	require.NoError(t, err)
	assert.NotContains(t, string(payload), "credentialRef")
	assert.NotContains(t, string(payload), "secret-id")
	assert.NotContains(t, string(payload), "secret-value")
}
