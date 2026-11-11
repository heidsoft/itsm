package skill_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/handlers/ai"
	"itsm-backend/handlers/skill"
)

// skill 域注册表实现归位回归锁（2026-11-11）。
//
// `service/skill_registry.go` 的实现本体已迁入 `handlers/skill/registry.go`：
// 旧结构下 skill.SkillRegistry / skill.NewSkillRegistry 不在域包，本文件
// 无法编译——这是「实现回到共享层」的第一道编译期防线。
var (
	// 域内窄契约（管理 API 的真实消费面）。
	_ skill.SkillRegistryCatalog = (*skill.SkillRegistry)(nil)
	// AI 评估侧的 dto 读面窄端口（bySkill 聚合只调 Get）。
	_ dto.SkillMetadataSource = (*skill.SkillRegistry)(nil)
	// 内置技能注册面（handlers/ai 的窄契约）。
	_ ai.SkillRegistrar = (*skill.SkillRegistry)(nil)
)

// TestSkillRegistryImplIsDomainOwned：实现文件必须在域包内且零共享层 import；
// service/ 顶层不得复活 skill_registry.go / skill_base.go / skill_manifest.go。
func TestSkillRegistryImplIsDomainOwned(t *testing.T) {
	dir := filepath.Join("..", "skill")
	body, err := os.ReadFile(filepath.Join(dir, "registry.go"))
	if err != nil {
		t.Fatalf("读取 handlers/skill/registry.go 失败（实现被移出域包？）：%v", err)
	}
	if len(body) < 1000 {
		t.Fatalf("扫描器零命中保护：handlers/skill/registry.go 仅 %d 字节", len(body))
	}
	if strings.Contains(string(body), `"itsm-backend/service"`) {
		t.Errorf("handlers/skill/registry.go 不得 import 共享层包")
	}

	// 旧文件名在 service/ 顶层复活即违规（L.1 基线已删除这 3 行）。
	for _, fn := range []string{"skill_registry.go", "skill_base.go", "skill_manifest.go"} {
		if _, err := os.Stat(filepath.Join("..", "..", "service", fn)); err == nil {
			t.Errorf("service/%s 复活：注册表实现唯一实现在 handlers/skill/registry.go", fn)
		}
	}

	// 域包构造器存在性（防止实现被抽象接口完全顶替后丢掉装配入口）。
	if !strings.Contains(string(body), "func NewSkillRegistry() *SkillRegistry") {
		t.Errorf("handlers/skill/registry.go 缺少 NewSkillRegistry 构造器")
	}
}
