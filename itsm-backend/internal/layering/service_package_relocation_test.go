package layering

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// service 层迁移收口专项回归锁（2026-11-12）。
//
// 本批把 service/ 顶层包（125 生产 + 207 测试 = 332 个 .go）整体 git mv 进
// internal/services/，包名仍是 service，全仓 136 处精确 import 换成新路径；
// service/ 的 8 个子包（cloud / bpmn / sla 等）保持原位，import 路径不变。
//
// 为什么是「整包归位」而不是按域拆分：go/ast 全量双向引用分析显示 332 个文件
// 的无向连通分量只有 5 个，最大分量 329 文件（124 生产 + 205 测试）——包内
// 私有符号与共享测试 helper 使绝大多数文件必须同包。先按 G1/G2/G3 拆域必须
// 做大规模符号解耦，属后续架构深化；本批只做物理归位，零行为变更。
//
// 假绿风险全部落在「编译全绿但守卫已空转」：
//   - service/ 顶层被清空后，L.1 文件名棘轮与 L.5 零复活扫描都会「零命中 PASS」；
//   - 只扫旧 import 路径会让 L.3 在改路径后恒为 0；
//   - C.6.4 service_go_files 归零后不再约束新目录的规模；
//   - 归位过程中漏搬/改名/改内容，编译仍可能过，但版本账本已经漂移。
//
// 因此本锁同时覆盖：旧目录零 .go（含扫描器有效性）、新目录生产文件集合 +
// 非空下界 + 包名不变、全仓零旧 import + 新 import 有效性、子包仍在、以及
// 门禁脚本与两份基线的强度。新增 *_test.go 不误报（测试增量是好事）。
// 内容完整性的一次性证据是本提交 332 个 R100 rename 与报告里的 SHA256 摘要；
// 不把内容哈希做成长期锚点，否则正常改一行业务代码（或并行会话 WIP）都要
// 同步改常量，守卫会退化成噪音。

const (
	sprProdCount        = 125
	sprTestCount        = 207
	sprMinProdBytes     = 200
	sprMinGoFiles       = 300
	sprMinNewImporters  = 100
	sprLegacySubdirsMin = 8
)

func sprBackendDir() string { return filepath.Join("..", "..") }
func sprRepoDir() string    { return filepath.Join("..", "..", "..") }

func sprReadRepo(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(append([]string{sprRepoDir()}, parts...)...)
	body, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("读取 %s 失败：%v", p, err)
	}
	if len(body) < 100 {
		t.Fatalf("扫描器零命中保护：%s 仅 %d 字节", p, len(body))
	}
	return string(body)
}

// TestSPRLegacyServiceDirHasNoTopLevelGo 守「归位后旧目录零 .go」。
// 同时要求 service/ 目录本身与 >=8 个子包仍在：整个目录被删掉时，
// 「零命中」同样不可信，且子包 import 路径会全仓断裂。
func TestSPRLegacyServiceDirHasNoTopLevelGo(t *testing.T) {
	dir := filepath.Join(sprBackendDir(), "service")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("service/ 目录消失（8 个子包应保持原位）：%v", err)
	}
	subdirs := 0
	var goFiles []string
	for _, e := range entries {
		if e.IsDir() {
			subdirs++
		} else if strings.HasSuffix(e.Name(), ".go") {
			goFiles = append(goFiles, e.Name())
		}
	}
	if subdirs < sprLegacySubdirsMin {
		t.Fatalf("扫描器零命中保护：service/ 子目录仅 %d 个（预期 >=%d）", subdirs, sprLegacySubdirsMin)
	}
	if len(goFiles) != 0 {
		t.Errorf("service/ 顶层重新出现 %d 个 .go（共享实现只许在 internal/services/）：%v", len(goFiles), goFiles)
	}
}

// TestSPRRelocatedProductionFilesAreIntact 锁 125 个生产文件的文件名集合、
// 包名与非空下界。内容完整性由本提交的 332 个 R100 rename 与
// output/service-layer-relocation-2026-11-12.md 中的一次性 SHA256 摘要证明：
// 不把内容哈希做成长期锚点，否则任何人正常改一行业务代码都要同步改常量，
// 守卫反而会变成噪音（并行会话 WIP 尤其容易触发）。
// 测试文件只锁数量下界，允许新增。
func TestSPRRelocatedProductionFilesAreIntact(t *testing.T) {
	dir := filepath.Join(sprBackendDir(), "internal", "services")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("internal/services/ 归位目录不可读：%v", err)
	}
	var prod, tests []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if strings.HasSuffix(e.Name(), "_test.go") {
			tests = append(tests, e.Name())
		} else {
			prod = append(prod, e.Name())
		}
	}
	sort.Strings(prod)
	if len(prod) != sprProdCount {
		t.Errorf("归位生产文件 %d 个（预期 %d）", len(prod), sprProdCount)
	}
	if len(tests) < sprTestCount {
		t.Errorf("归位测试文件仅 %d 个（预期 >=%d，漏搬会让回归锁静默消失）", len(tests), sprTestCount)
	}
	for _, name := range prod {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("读取 internal/services/%s 失败：%v", name, err)
		}
		if len(body) < sprMinProdBytes {
			t.Errorf("internal/services/%s 仅 %d 字节（占位/清空/漏搬都会让实现静默消失）", name, len(body))
		}
		// 少数文件带 // test-coverage-guard: skip 之类的首行注释，不能只判首行。
		if !strings.Contains(string(body), "\npackage service\n") &&
			!strings.HasPrefix(string(body), "package service\n") {
			t.Errorf("internal/services/%s 包名漂移（必须仍是 package service，否则 136 处限定符失配）", name)
		}
	}

	// 生产文件名集合必须逐行等于 L.1 棘轮清单：新增/改名/漏搬由 C.8 与此处双重挡下。
	base := sprReadRepo(t, "scripts", "docs-gate", "service-prod-baseline.txt")
	var baseNames []string
	for _, line := range strings.Split(base, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		baseNames = append(baseNames, line)
	}
	sort.Strings(baseNames)
	if strings.Join(prod, "\n") != strings.Join(baseNames, "\n") {
		t.Errorf("归位生产文件集合与 L.1 基线不一致：prod=%d baseline=%d", len(prod), len(baseNames))
	}
}

// TestSPRImportsPointOnlyToRelocatedPackage 全仓扫描精确 import 字面量。
// needle 用拼接构造，避免本锁自身成为命中项（扫描器自证）。
func TestSPRImportsPointOnlyToRelocatedPackage(t *testing.T) {
	oldNeedle := "\"itsm-backend/" + "service\""
	newNeedle := "\"itsm-backend/internal/" + "services\""
	root := sprBackendDir()
	scanned := 0
	var oldHits, newHits []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		scanned++
		s := string(body)
		if strings.Contains(s, oldNeedle) {
			oldHits = append(oldHits, p)
		}
		if strings.Contains(s, newNeedle) {
			newHits = append(newHits, p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("全仓 .go 扫描失败：%v", err)
	}
	if scanned < sprMinGoFiles {
		t.Fatalf("扫描器零命中保护：只扫到 %d 个 .go（预期 >=%d）", scanned, sprMinGoFiles)
	}
	if len(oldHits) > 0 {
		t.Errorf("全仓仍有 %d 个文件 import 旧共享层路径：%v", len(oldHits), oldHits[:sprMin(len(oldHits), 5)])
	}
	if len(newHits) < sprMinNewImporters {
		t.Errorf("新共享层路径只被 %d 个文件引用（预期 >=%d，扫描器或批量替换失效）", len(newHits), sprMinNewImporters)
	}
}

func sprMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestSPRGatesStayStrongAfterRelocation 防止「把守卫一起改松」造成账面归零。
func TestSPRGatesStayStrongAfterRelocation(t *testing.T) {
	c8 := sprReadRepo(t, "scripts", "docs-gate", "check-layering-convergence.sh")
	checks := map[string]string{
		"L.1b 旧目录零 .go": "collect_legacy_service_go",
		"L.1 扫描器有效性":    `"${l1_files}" -lt 100`,
		"L.3 新旧路径双扫":    `-e '"itsm-backend/internal/services"'`,
		"L.5 改扫归位目录":    `find "${BE}/internal/services" -maxdepth 1 -name '*.go'`,
		"L.1 棘轮改扫归位目录":  `find "${BE}/internal/services" -maxdepth 1 -name '*.go' -not`,
		"C.8 L.1b 调用点":  `legacy_go="$(collect_legacy_service_go)"`,
	}
	for label, needle := range checks {
		if !strings.Contains(c8, needle) {
			t.Errorf("C.8 守卫被削弱或改回旧实现：%s 缺失（needle=%s）", label, needle)
		}
	}
	c6 := sprReadRepo(t, "scripts", "docs-gate", "check-product-drift.sh")
	if !strings.Contains(c6, "shared_services_go_files)") {
		t.Error("C.6.4 缺少归位目录上界棘轮 measure（旧键归零后新目录将免检）")
	}
	pb := sprReadRepo(t, "scripts", "docs-gate", "product-surface-baseline.txt")
	for _, needle := range []string{"service_go_files=0", "shared_services_go_files=332"} {
		if !strings.Contains(pb, needle) {
			t.Errorf("C.6.4 基线缺少 %q（不得留旧上界或漏登记新目录）", needle)
		}
	}
}

// TestSPRServiceSubpackagesStayPut 子包 import 路径未随本批改动；误搬会让
// 47+41+22 处 "itsm-backend/service/<sub>" 全部失效。
func TestSPRServiceSubpackagesStayPut(t *testing.T) {
	for _, name := range []string{"approver", "bpmn", "cloud", "common", "lifecycle", "marketplace", "scenario", "sla"} {
		dir := filepath.Join(sprBackendDir(), "service", name)
		count := 0
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !d.IsDir() && strings.HasSuffix(p, ".go") {
				count++
			}
			return nil
		})
		if err != nil || count == 0 {
			t.Errorf("service/%s 子包丢失或零 .go（err=%v, count=%d）", name, err, count)
		}
	}
}
