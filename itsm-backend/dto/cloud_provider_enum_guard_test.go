package dto

// 本文件是「云厂商全集」的单一真源守卫。
//
// 背景（实测发现，不是假想）：provider 枚举在 dto 里以 binding oneof 硬编码了 4 份
// （cloud_dto 两个列表查询 + cmdb_dto 云账号/云服务两个写入体），值还是 2026 年初的
// 六个：aliyun/tencent/huawei/aws/azure/onprem。而 service/cloud 在 2026-10-24 已经
// 接入了 volcengine 与 k8s 两个发现适配器——结果是**适配器写好了、产品面根本选不到**：
// 建云账号时 provider=volcengine 会被 400 拒掉，发现能力等于零。
//
// 「适配器已接入」与「账号能建」分属两处清单，这类漂移不会有任何编译错误，只会表现为
// 「功能做完了但用户用不了」。所以这里强制两件事：
//  1. dto 里每一处 provider oneof 必须等于 cloud.KnownProviders（多一个少一个都红）；
//  2. KnownProviders 里每个厂商都必须在 ProviderDisplayName 有中文名，
//     否则下拉框会直接显示裸标识。
//
// 采用源码扫描而不是反射枚举结构体：反射要求人工维护「要检查哪些结构体」的清单，
// 那份清单本身就会漂移——漏登记的结构体恰好是下一次出错的地方。

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"itsm-backend/service/cloud"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// providerOneofRe 抓取 binding tag 里的 oneof 取值段。
// 只匹配到下一个空格/反引号/双引号为止，因此不会把后面的 comment:"..." 吞进来。
var providerOneofRe = regexp.MustCompile(`oneof=([a-z0-9 ]+)`)

// dtoDir 定位本包目录，与调用方 cwd 无关。
func dtoDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "无法定位本测试文件位置")
	return filepath.Dir(thisFile)
}

// scanProviderEnums 扫描 dto 包源码里所有「看起来是云厂商枚举」的 oneof 取值。
// 判定标准是取值段里含 aliyun：本包没有第二个含 aliyun 的枚举。
//
// 返回 文件 → 该文件中每一处枚举的取值集合（不是并集）：同一文件里写两处枚举是
// 常态（列表查询 + 写入体），并集会掩盖「其中一处漏改」。
func scanProviderEnums(t *testing.T) map[string][][]string {
	t.Helper()
	dir := dtoDir(t)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	found := make(map[string][][]string)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		require.NoError(t, err)
		for _, m := range providerOneofRe.FindAllStringSubmatch(string(raw), -1) {
			tokens := strings.Fields(m[1])
			isProvider := false
			for _, tok := range tokens {
				if tok == "aliyun" {
					isProvider = true
					break
				}
			}
			if !isProvider {
				continue
			}
			found[e.Name()] = append(found[e.Name()], tokens)
		}
	}
	return found
}

func TestProviderEnumMatchesKnownProviders(t *testing.T) {
	want := append([]string(nil), cloud.KnownProviders...)
	sort.Strings(want)

	enums := scanProviderEnums(t)
	// 兜底：扫描器必须真的找到东西。若哪天有人把 oneof 换成别的写法，这里的
	// 「0 处」会让下面的循环空转、测试假绿——那正是守卫最容易失效的方式。
	require.NotEmpty(t, enums, "未在 dto 包源码里找到任何 provider oneof 枚举，守卫已失效，请修正扫描器或改回 oneof 写法")

	total := 0
	for file, occurrences := range enums {
		for i, tokens := range occurrences {
			total++
			got := append([]string(nil), tokens...)
			sort.Strings(got)
			assert.Equal(t, want, got,
				"%s 第 %d 处 provider 枚举与 cloud.KnownProviders 不一致：新增/移除厂商必须同步改这里，否则新接入的适配器在产品面选不到",
				file, i+1)
		}
	}
	// 4 处枚举是 2026-10-24 实测到的现状；新增一处而没同步改值会被上面的断言拦下，
	// 这里再钉一下数量是为了让「有人把某处枚举整个删掉」也变红。
	assert.Equal(t, 4, total, "dto 里 provider 枚举的处数变了，请确认是刻意增删还是误删")
}

func TestKnownProvidersHaveDisplayNames(t *testing.T) {
	for _, p := range cloud.KnownProviders {
		name := cloud.ProviderDisplayName(p)
		assert.NotEmpty(t, name, "provider %q 缺显示名", p)
		// ProviderDisplayName 对未登记厂商回退为原样返回，等于「下拉框显示裸标识」。
		assert.NotEqual(t, p, name, "provider %q 未登记显示名，下拉框会显示裸标识", p)
		assert.Equal(t, p, cloud.NormalizeProvider(p),
			"KnownProviders 必须存归一后的标识，否则与 DTO 枚举比对时永远不等", p)
	}
}

func TestIsKnownProvider(t *testing.T) {
	for _, p := range cloud.KnownProviders {
		assert.True(t, cloud.IsKnownProvider(p), "%q 应属于已知厂商", p)
	}
	// 别名归一后同样算已知：用户在别处填 alibaba 不应被判成厂商不存在。
	for _, alias := range []string{"alibaba", "alicloud", "qcloud", "tencentcloud", "huaweicloud", "amazon", "volc", "kubernetes"} {
		assert.True(t, cloud.IsKnownProvider(alias), "别名 %q 归一后应属于已知厂商", alias)
	}
	assert.False(t, cloud.IsKnownProvider("not-a-cloud"))
	assert.False(t, cloud.IsKnownProvider(""))
}

// TestProviderEnumGuardScansRealFiles 防止扫描器因路径定位失败而静默空转。
func TestProviderEnumGuardScansRealFiles(t *testing.T) {
	dir := dtoDir(t)
	_, err := os.Stat(filepath.Join(dir, "cloud_dto.go"))
	require.NoError(t, err, "守卫测试定位到的 dto 目录不对: "+dir)
	f, err := os.Open(filepath.Join(dir, "cloud_dto.go"))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	head := make([]byte, 64)
	_, err = io.ReadFull(f, head)
	require.NoError(t, err)
	assert.Contains(t, string(head), "package dto")
}
