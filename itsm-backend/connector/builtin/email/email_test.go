package email

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseInboundMultipartEmail(t *testing.T) {
	raw := []byte("From: Customer <customer@example.com>\r\nTo: noc@example.com\r\nMessage-ID: <m1@example.com>\r\nSubject: =?UTF-8?B?57q/6Lev5Lit5pat?=\r\nContent-Type: multipart/alternative; boundary=x\r\n\r\n--x\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nMPLS down\r\n--x\r\nContent-Type: text/html; charset=utf-8\r\n\r\n<b>MPLS down</b>\r\n--x--\r\n")
	message, err := parseInbound(raw)
	require.NoError(t, err)
	require.Equal(t, "Customer <customer@example.com>", message.UserID)
	require.Contains(t, message.Content, "MPLS down")
	require.Equal(t, "<m1@example.com>", message.Extras["externalMessageId"])
}

func TestParseInboundRejectsInvalidFromHeader(t *testing.T) {
	_, err := parseInbound([]byte("From: bad\r\nTo: noc@example.com\r\n\r\nbody"))
	require.Error(t, err)
}

func TestManifestDeclaresEmailCapabilities(t *testing.T) {
	manifest := New().Manifest()
	require.Equal(t, "email", manifest.Name)
	require.Contains(t, manifest.ConfigSchema, "imapHost")
	require.NotEmpty(t, manifest.RequiredPermissions)
}
