package email

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"itsm-backend/connector"
)

func TestParseInboundMultipartEmail(t *testing.T) {
	raw := []byte("From: Customer <customer@example.com>\r\nTo: noc@example.com\r\nMessage-ID: <m1@example.com>\r\nSubject: =?UTF-8?B?57q/6Lev5Lit5pat?=\r\nContent-Type: multipart/alternative; boundary=x\r\n\r\n--x\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nMPLS down\r\n--x\r\nContent-Type: text/html; charset=utf-8\r\n\r\n<b>MPLS down</b>\r\n--x--\r\n")
	message, err := parseInbound(raw)
	require.NoError(t, err)
	require.Equal(t, "Customer <customer@example.com>", message.UserID)
	require.Contains(t, message.Content, "MPLS down")
	require.Equal(t, "<m1@example.com>", message.Extras["externalMessageId"])
}

func TestInitRejectsPlaintextAndPrivateMailTargets(t *testing.T) {
	base := connector.Config{TenantID: 1, Name: "email", Provider: "standard", Enabled: true, Credentials: map[string]string{"username": "noc@example.com", "password": "secret"}}
	plain := base
	plain.Settings = map[string]interface{}{"imapHost": "imap.example.com", "imapPort": 993, "imapTlsMode": "plain", "smtpHost": "smtp.example.com", "smtpPort": 465, "smtpTlsMode": "ssl"}
	require.Error(t, New().Init(context.Background(), plain))

	privateTarget := base
	privateTarget.Settings = map[string]interface{}{"imapHost": "127.0.0.1", "imapPort": 993, "smtpHost": "169.254.169.254", "smtpPort": 465}
	require.Error(t, New().Init(context.Background(), privateTarget))

	carrierGradeNAT := base
	carrierGradeNAT.Settings = map[string]interface{}{"imapHost": "100.64.0.1", "imapPort": 993, "smtpHost": "smtp.example.com", "smtpPort": 465}
	require.Error(t, New().Init(context.Background(), carrierGradeNAT))
}

func TestParseInboundRejectsInvalidFromHeader(t *testing.T) {
	_, err := parseInbound([]byte("From: bad\r\nTo: noc@example.com\r\n\r\nbody"))
	require.Error(t, err)
}

func TestParseInboundRejectsAutomatedReplies(t *testing.T) {
	_, err := parseInbound([]byte("From: robot@example.com\r\nTo: noc@example.com\r\nAuto-Submitted: auto-replied\r\n\r\nbody"))
	require.Error(t, err)
}

func TestManifestDeclaresEmailCapabilities(t *testing.T) {
	manifest := New().Manifest()
	require.Equal(t, "email", manifest.Name)
	require.Contains(t, manifest.ConfigSchema, "imapHost")
	require.NotEmpty(t, manifest.RequiredPermissions)
}
