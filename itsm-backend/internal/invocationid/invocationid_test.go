package invocationid_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"itsm-backend/internal/authz"
	"itsm-backend/internal/invocationid"
)

func TestRoundTripAllPrincipals(t *testing.T) {
	cases := []struct {
		principal string
		userID    int
	}{
		{authz.PrincipalHuman, 7},
		{authz.PrincipalAgent, 7},
		{authz.PrincipalService, 0},
	}
	for _, tc := range cases {
		t.Run(tc.principal, func(t *testing.T) {
			raw, err := invocationid.Encode(map[string]interface{}{"page": float64(2)}, tc.principal)
			require.NoError(t, err)
			d, err := invocationid.Decode(raw, "some-fingerprint", tc.userID)
			require.NoError(t, err)
			require.Equal(t, tc.principal, d.Principal)
			require.Equal(t, invocationid.SourceStored, d.Source)
			require.Equal(t, float64(2), d.Args["page"])
			_, leaked := d.Args["_itsmInvocation"]
			require.False(t, leaked, "信封不得泄漏给工具")
		})
	}
}

func TestEncodeOverridesClientSuppliedEnvelope(t *testing.T) {
	raw, err := invocationid.Encode(map[string]interface{}{
		"_itsmInvocation": map[string]interface{}{"version": 1, "principal": "service"},
	}, authz.PrincipalHuman)
	require.NoError(t, err)
	d, err := invocationid.Decode(raw, "", 7)
	require.NoError(t, err)
	require.Equal(t, authz.PrincipalHuman, d.Principal, "客户端伪造的 principal 必须被服务端覆盖")
}

func TestDecodeLegacy(t *testing.T) {
	// 无信封 + 正数 userID + 非空指纹 → 兼容推断为 agent。
	d, err := invocationid.Decode(`{"page":1}`, "fingerprint", 7)
	require.NoError(t, err)
	require.Equal(t, authz.PrincipalAgent, d.Principal)
	require.Equal(t, invocationid.SourceInferred, d.Source)

	// 无信封 + 正数 userID + 空指纹 → human。
	d, err = invocationid.Decode(`{"page":1}`, "", 7)
	require.NoError(t, err)
	require.Equal(t, authz.PrincipalHuman, d.Principal)

	// userID=0 的历史记录无法安全还原 → fail-closed。
	_, err = invocationid.Decode(`{"page":1}`, "fingerprint", 0)
	require.Error(t, err)
	require.True(t, errors.Is(err, invocationid.ErrInvalid))
}

func TestDecodeFailClosed(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"bad json", `not-json`},
		{"unknown version", `{"_itsmInvocation":{"version":99,"principal":"human"}}`},
		{"unknown principal", `{"_itsmInvocation":{"version":1,"principal":"root"}}`},
		{"service with user id", `{"_itsmInvocation":{"version":1,"principal":"service"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := invocationid.Decode(tc.raw, "", 7)
			require.Error(t, err)
			require.True(t, errors.Is(err, invocationid.ErrInvalid))
		})
	}
}
