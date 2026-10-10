package marketplace

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRestoreMaskedConfigRoundTripPreservesRealSecrets(t *testing.T) {
	stored := map[string]interface{}{
		"oauth": map[string]interface{}{
			"access_token":  "real-access",
			"refresh_token": "real-refresh",
			"expires_in":    float64(7200),
		},
		"settings": map[string]interface{}{
			"region": "cn-north",
		},
	}
	incoming := map[string]interface{}{
		"oauth": map[string]interface{}{
			"access_token":  "***",
			"refresh_token": "***",
			"expires_in":    float64(7200),
		},
		"settings": map[string]interface{}{
			"region": "cn-north",
		},
	}

	restored := RestoreMaskedConfig(incoming, stored)

	require.Equal(t, "real-access", restored["oauth"].(map[string]interface{})["access_token"])
	require.Equal(t, "real-refresh", restored["oauth"].(map[string]interface{})["refresh_token"])
	require.Equal(t, float64(7200), restored["oauth"].(map[string]interface{})["expires_in"])
	require.Equal(t, "cn-north", restored["settings"].(map[string]interface{})["region"])
}

func TestRestoreMaskedConfigKeepsUserEditedValues(t *testing.T) {
	stored := map[string]interface{}{
		"credentials": map[string]interface{}{
			"appId": "cli_old",
		},
	}
	incoming := map[string]interface{}{
		"credentials": map[string]interface{}{
			"appId": "cli_new",
		},
		"extra": map[string]interface{}{
			"newKey": "newValue",
		},
	}

	restored := RestoreMaskedConfig(incoming, stored)

	require.Equal(t, "cli_new", restored["credentials"].(map[string]interface{})["appId"])
	require.Equal(t, "newValue", restored["extra"].(map[string]interface{})["newKey"])
}

func TestRestoreMaskedConfigNilIncoming(t *testing.T) {
	require.Nil(t, RestoreMaskedConfig(nil, map[string]interface{}{"a": "b"}))
}
