package marketplace

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaskSensitiveConfigRedactsNestedSecretsAndKeepsStructure(t *testing.T) {
	config := map[string]interface{}{
		"credentials": map[string]interface{}{
			"appId":             "cli_123",
			"appSecret":         "real-secret",
			"verificationToken": "real-token",
		},
		"oauth": map[string]interface{}{
			"access_token":  "real-access",
			"refresh_token": "real-refresh",
			"expires_in":    float64(7200),
		},
		"settings": map[string]interface{}{
			"region": "cn-north",
		},
	}

	masked := MaskSensitiveConfig(config)

	require.Equal(t, "***", masked["credentials"].(map[string]interface{})["appSecret"])
	require.Equal(t, "***", masked["credentials"].(map[string]interface{})["verificationToken"])
	require.Equal(t, "cli_123", masked["credentials"].(map[string]interface{})["appId"])
	require.Equal(t, "***", masked["oauth"].(map[string]interface{})["access_token"])
	require.Equal(t, "***", masked["oauth"].(map[string]interface{})["refresh_token"])
	require.Equal(t, float64(7200), masked["oauth"].(map[string]interface{})["expires_in"])
	require.Equal(t, "cn-north", masked["settings"].(map[string]interface{})["region"])

	serialized, _ := json.Marshal(masked)
	require.NotContains(t, string(serialized), "real-secret")
	require.NotContains(t, string(serialized), "real-access")
	require.NotContains(t, string(serialized), "real-refresh")
}

func TestMaskSensitiveConfigHandlesNilAndNonMapValues(t *testing.T) {
	require.Nil(t, MaskSensitiveConfig(nil))
	masked := MaskSensitiveConfig(map[string]interface{}{"count": float64(3)})
	require.Equal(t, float64(3), masked["count"])
}
