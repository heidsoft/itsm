package connector_test

import (
	"strings"
	"testing"

	"itsm-backend/connector"

	// 触发所有内置连接器的 init() 注册
	_ "itsm-backend/connector/builtin/console"
	_ "itsm-backend/connector/builtin/dingtalk"
	_ "itsm-backend/connector/builtin/feishu"
	_ "itsm-backend/connector/builtin/webhook"
	_ "itsm-backend/connector/builtin/wecom"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOfficialManifestCoverage 是发布认证 P7 的静态覆盖门禁：
// 所有已注册的官方连接器 manifest 必须具备 version、checksum、权限声明。
func TestOfficialManifestCoverage(t *testing.T) {
	manifests := connector.Default().List()
	require.NotEmpty(t, manifests, "至少应注册一个内置连接器")

	expected := []string{"console", "dingtalk", "feishu", "webhook", "wecom"}
	found := make(map[string]bool, len(manifests))

	for _, m := range manifests {
		found[m.Name] = true
		t.Run(m.Name, func(t *testing.T) {
			assert.NotEmpty(t, m.Version, "version 必须声明")
			assert.NotEmpty(t, m.RequiredPermissions, "required_permissions 必须声明")
			assert.True(t, strings.HasPrefix(m.Checksum, "sha256:"), "checksum 必须由注册表计算")
			assert.Equal(t, m.ComputeChecksum(), m.Checksum, "checksum 必须与身份字段一致")
			assert.True(t, m.IsOfficial, "内置连接器必须标记为官方组件")
		})
	}

	for _, name := range expected {
		assert.True(t, found[name], "官方连接器 %s 必须在注册表中", name)
	}
}

// TestManifestChecksumDeterministic 校验和必须确定且对安全字段敏感。
func TestManifestChecksumDeterministic(t *testing.T) {
	m := connector.Manifest{
		Name:                "demo",
		Version:             "1.0.0",
		Provider:            "demo",
		Type:                connector.TypeCustom,
		RequiredPermissions: []string{"connector:write"},
	}
	assert.Equal(t, m.ComputeChecksum(), m.ComputeChecksum(), "同一 manifest 校验和必须稳定")

	changed := m
	changed.Version = "1.0.1"
	assert.NotEqual(t, m.ComputeChecksum(), changed.ComputeChecksum(), "版本变化必须改变校验和")

	permChanged := m
	permChanged.RequiredPermissions = []string{"connector:write", "ticket:write"}
	assert.NotEqual(t, m.ComputeChecksum(), permChanged.ComputeChecksum(), "权限变化必须改变校验和")

	// 展示类字段不影响校验和
	cosmetic := m
	cosmetic.Description = "另一个描述"
	assert.Equal(t, m.ComputeChecksum(), cosmetic.ComputeChecksum(), "展示字段不应影响校验和")
}

// TestRegisterRejectsIncompleteManifest 注册门禁必须 fail closed。
func TestRegisterRejectsIncompleteManifest(t *testing.T) {
	tests := []struct {
		name     string
		manifest connector.Manifest
	}{
		{"缺少 version", connector.Manifest{Name: "x", RequiredPermissions: []string{"connector:write"}}},
		{"缺少权限声明", connector.Manifest{Name: "x", Version: "1.0.0"}},
		{"缺少 name", connector.Manifest{Version: "1.0.0", RequiredPermissions: []string{"connector:write"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, tt.manifest.ValidateForRegistration())
		})
	}
}
