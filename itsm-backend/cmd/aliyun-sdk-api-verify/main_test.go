package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// readOnlyAPIPattern 扫描所有 client.*WithOptions( 调用。SDK 的每个 WithOptions
// 方法对应一次真实的云 API 调用，因此这个清单就是「本命令会打到阿里云的
// 全部动作」，用它做白名单比逐个 review 闭包可靠。
var readOnlyAPIPattern = regexp.MustCompile(`client\.([A-Za-z0-9]+)WithOptions\(`)

// writeAPIAllowlist 是允许出现的非 Describe 调用。只允许 RunInstances，
// 且必须经由 DryRun 路径（由下一组断言锁定）。
var writeAPIAllowlist = map[string]bool{"RunInstances": true}

func scanAPIcalls(src string) (map[string]int, error) {
	matches := readOnlyAPIPattern.FindAllStringSubmatch(src, -1)
	if len(matches) == 0 {
		return nil, errNoAPICall
	}
	counts := make(map[string]int, len(matches))
	for _, m := range matches {
		counts[m[1]]++
	}
	return counts, nil
}

var errNoAPICall = &scanError{"源码里没有扫描到任何 client.*WithOptions 调用，扫描器或调用方式已漂移"}

type scanError struct{ msg string }

func (e *scanError) Error() string { return e.msg }

// assertNoWriteAPI 是本命令的安全契约：
//  1. 扫描器必须命中调用（零命中直接失败，避免守卫假绿）；
//  2. 每个调用必须是 Describe*，或是白名单里的 RunInstances；
//  3. RunInstances 只能走 DryRun：所有 RunInstancesWithOptions 调用点附近
//     必须出现 DryRun=true 或来自 BuildDryRunRequest 的请求；
//  4. 源码里不得出现任何会创建/删除/启停资源的 SDK 方法名。
func assertNoWriteAPI(src string) error {
	counts, err := scanAPIcalls(src)
	if err != nil {
		return err
	}
	for api, n := range counts {
		if strings.HasPrefix(api, "Describe") {
			continue
		}
		if !writeAPIAllowlist[api] {
			return &scanError{"出现非只读的云 API 调用: " + api + "（" + itoa(n) + " 处）"}
		}
	}

	// RunInstances 的每一处调用都必须能追溯到 DryRun=true。
	// 按调用点起始位置逐处切窗口判定：Go 参数里嵌套 tea.String(...)，用正则
	// 匹配到第一个右括号会把窗口截断，反而漏判（守卫自身的假绿比没有守卫更危险）。
	// 逐处比对而不是「源码里只要有 DryRun 就算过」，避免后来新增一处真写调用
	// 被别处已有的 DryRun 字样掩盖。
	const callMarker = "client.RunInstancesWithOptions("
	runCount := 0
	for off := 0; ; {
		idx := strings.Index(src[off:], callMarker)
		if idx < 0 {
			break
		}
		start := off + idx
		end := min(start+1400, len(src))
		window := src[start:end]
		runCount++
		reusesDryRunRequest := strings.Contains(window, "dryReq")
		explicitDryRun := strings.Contains(window, "DryRun:") && strings.Contains(window, "tea.Bool(true)")
		if !reusesDryRunRequest && !explicitDryRun {
			return &scanError{"RunInstances 调用未走 DryRun=true 路径（第 " + itoa(runCount) + " 处）"}
		}
		off = start + len(callMarker)
	}
	if runCount != counts["RunInstances"] {
		return &scanError{"RunInstances 调用点数与扫描结果不一致，守卫无法逐处判定"}
	}
	// 必须匹配调用形式而非裸标识符：文件头的说明注释里也写着
	// BuildDryRunRequest，只查字符串会被注释满足，守卫对「调用被删」假绿。
	if !strings.Contains(src, "alicloud.BuildDryRunRequest(") {
		return &scanError{"DryRun 请求必须调用 alicloud.BuildDryRunRequest，不得自行拼参数"}
	}
	if !strings.Contains(src, "tea.Bool(true)") {
		return &scanError{"DryRun 必须显式置 true"}
	}

	// 反向清单：任何会真正改变资源/产生费用的方法名都不许出现。
	forbidden := []string{
		"CreateInstanceWithOptions", "DeleteInstanceWithOptions", "StartInstanceWithOptions",
		"StopInstanceWithOptions", "RebootInstanceWithOptions", "ReleaseInstanceWithOptions",
		"ModifyInstanceAttributeWithOptions", "ReplaceSystemDiskWithOptions",
		"AllocatePublicIpAddressWithOptions", "CreateDiskWithOptions", "DeleteDiskWithOptions",
		"CreateSecurityGroupWithOptions", "DeleteSecurityGroupWithOptions",
		"CreateVSwitchWithOptions", "DeleteVSwitchWithOptions", "RunCommandWithOptions",
	}
	for _, name := range forbidden {
		if strings.Contains(src, name) {
			return &scanError{"出现会产生费用或改变资源的写 API: " + name}
		}
	}
	return nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

const sourcePath = "main.go"

func TestVerifyCommandHasNoWriteAPI(t *testing.T) {
	src, err := os.ReadFile(sourcePath)
	require.NoError(t, err)
	require.NoError(t, assertNoWriteAPI(string(src)))

	// 扫描器零命中必须失败：否则将来调用方式一改，守卫会静默变绿。
	_, err = scanAPIcalls("package main\n// no calls here\n")
	require.Error(t, err, "扫描器零命中必须判失败")
	require.NoError(t, assertNoWriteAPI(string(src)), "真实源码仍须通过")
}

func TestVerifyCommandRejectsInjectedWriteAPI(t *testing.T) {
	src, err := os.ReadFile(sourcePath)
	require.NoError(t, err)
	base := string(src)

	// 反向验证一：把一处只读调用换成真写调用，守卫必须红。
	injected := strings.Replace(base, "client.DescribeRegionsWithOptions(", "client.CreateInstanceWithOptions(", 1)
	require.Error(t, assertNoWriteAPI(injected), "注入写 API 必须被拒")

	// 反向验证二：让 DryRun 不再是复用生产构造逻辑，守卫必须红。
	// 只替换调用点，注释里的同名标识符保留——若守卫只查裸字符串就会假绿。
	noDryRun := strings.Replace(base, "alicloud.BuildDryRunRequest(", "manualBuild(", 1)
	require.Error(t, assertNoWriteAPI(noDryRun), "绕开 buildDryRunRequest 调用必须被拒")

	// 反向验证三：把显式的 DryRun=true 改掉，守卫必须红。
	noTrue := strings.Replace(base, "DryRun:             tea.Bool(true),", "DryRun:             tea.Bool(false),", 1)
	require.Error(t, assertNoWriteAPI(noTrue), "DryRun 置 false 必须被拒")
}
