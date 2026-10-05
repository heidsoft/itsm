package contract

// 错误消息泄漏棘轮（ratchet）。
//
// AGENTS.md「Context、日志与可观测性」要求：原始 provider/driver 错误只能进服务端日志，
// 不得作为响应消息透出。common.FailWithErr / NotFoundWithErr / BindValidationError 这类
// *WithErr helper 就是为此存在的安全包装；直接把 err.Error() 当公共消息传给
// common.Fail*/NotFound*/InternalError* 属于泄漏面（SQL 片段、ent 校验串、连接器报文、
// 服务端落盘路径都可能出现在响应里）。
//
// 本文件不要求一次性清零存量债务，但：
//   - 禁止新增泄漏点（新文件、或某文件计数上升都会红）；
//   - 计数下降同样要红一次，迫使提交者把基线改小并写进 CHANGELOG，
//     避免「悄悄少了几处却没人知道」。
//
// 收敛一处泄漏的做法：改用 common.FailWithErr(c, err, "面向用户的安全文案")；
// 若领域层已返回 common.AppError/BusinessError，则改用 common.RespondError，让
// common.classifyError 决定状态码而不是把业务拒绝压成 500。同时给该端点补
// status + 业务 code + 「响应体不含原始错误串」的契约断言，最后把基线里该文件计数改小。
//
// 跑测命令：cd itsm-backend && go test ./tests/contract/... -run TestErrorLeakRatchet
// 重新采集基线：ERROR_LEAK_RATCHET_DUMP=1 go test ./tests/contract/... -run TestErrorLeakRatchet -v

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// errorLeakScanDirs 是扫描范围：只有进入 HTTP 边界的代码会把错误串写进响应。
// service/ 内部把 err.Error() 拼进日志或包装错误属于诊断用途，不在本棘轮范围。
// legacy controller/ 目录实测已不存在（HTTP facade 已清空），因此不列入；
// 缺失的目录会被跳过而不是让棘轮自己变红。
var errorLeakScanDirs = []string{"handlers", "router"}

// failHelperRE 匹配「把错误串当公共消息写进响应」的调用形态：同一行里同时出现
// common.<fail 系 helper>( 和 err.Error()。项目里这类调用实测都是单行；
// 若将来出现多行形态，应先扩展规则并保持基线可复现，而不是绕过棘轮。
var failHelperRE = regexp.MustCompile(`common\.(Fail|FailWithData|ParamError|ValidationErrorResponse|AuthFailed|Forbidden|NotFound|InternalError)[A-Za-z]*\(.*err\.Error\(\)`)

// errorLeakBaseline 是 2026-10-04 由本文件扫描器实测的每文件泄漏计数（合计 413 处 / 35 个文件）。
// handlers/notification/handler.go 29，收敛应从它们按文件整片推进。
//
// router/ticket_routes.go 原有 3 处已清零：工单关联的三个读取端点改走
// common.RespondError，业务拒绝由 service 的 common.BusinessError 决定状态码。
//
// handlers/cmdb/handler.go 原有 2 处已清零（本文件从基线移除）：对账读取与云选择器的
// 绑定失败不再拼 err.Error()，同时域内 15 条云/发现端点统一由 RespondError 分类，
// 域内私有的 failCMDBError 映射随之删除。
// 前三名占了近半数：handlers/bpmn/workflow.go 36（2026-10-05 错误净化收敛）、email_intake 27。
// production_service.go 73 已在 2026-10-05 RespondError+ParamError sweep 中清零并从此基线删除；
// handlers/notification/handler.go 29 已在 2026-10-05 RespondError+ParamError+AuthFailed sweep 中清零并从此基线删除；
// handlers/email_intake/handler.go 27 已在 2026-10-05 RespondError+NotFound sweep 中清零并从此基线删除；
// handlers/change/handler.go 20 已在 2026-10-05 ParamErrorWithErr/RespondError/FailWithErr sweep 中清零并从此基线删除；
// handlers/common/handler.go 22 已在 2026-10-06 ParamErrorWithErr/RespondError/AuthFailed+logger sweep 中清零并从此基线删除；
// handlers/sla/handler.go 20 已在 2026-10-06 ParamErrorWithErr/RespondError sweep 中清零并从此基线删除。
var errorLeakBaseline = map[string]int{
	"handlers/bpmn/ai_generator.go":         2,
	"handlers/bpmn/dashboard.go":            1,
	"handlers/bpmn/lint.go":                 2,
	"handlers/bpmn/process_trigger.go":      6, // 2026-10-05 RespondError 收敛
	"handlers/bpmn/workflow.go":             36,
	"handlers/bpmn/workflow_template.go":    3,
	"handlers/application/handler.go":       12,
	"handlers/approval/routes.go":           12,
	"handlers/approval_chain/handler.go":    2,
	"handlers/auditlog/handler.go":          2,
	"handlers/auth/handler.go":              10,
	"handlers/connector/handler.go":         10,
	"handlers/escalation_matrix/handler.go": 1,
	"handlers/feishu/handler.go":            1,
	"handlers/incident/handler.go":          1,
	"handlers/knowledge/handler.go":         7,
	"handlers/known_error/handler.go":       2,
	"handlers/marketplace/handler.go":       13,
	"handlers/prediction/handler.go":        4,
	"handlers/project/handler.go":           7,
	"handlers/rbac/handler.go":              14,
	"handlers/sla_template/handler.go":      2,
	"handlers/standard_change/handler.go":   2,
	"handlers/survey/handler.go":            7,
	"handlers/systemconfig/routes.go":       2,
	"handlers/user/handler.go":              2,
	"handlers/workbench/handler.go":         1,
	"router/dashboard_routes.go":            2,
}

// measuredErrorLeaks 扫描真实源码，返回「相对 itsm-backend 的路径 -> 泄漏计数」。
func measuredErrorLeaks(t *testing.T) map[string]int {
	t.Helper()

	backendRoot := filepath.Join("..", "..")
	counts := map[string]int{}
	for _, dir := range errorLeakScanDirs {
		root := filepath.Join(backendRoot, dir)
		if _, err := os.Stat(root); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatalf("检查扫描目录 %s 失败: %v", dir, err)
		}

		err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			rel, relErr := filepath.Rel(backendRoot, path)
			if relErr != nil {
				return relErr
			}
			n := 0
			for _, line := range strings.Split(string(data), "\n") {
				if failHelperRE.MatchString(line) {
					n++
				}
			}
			if n > 0 {
				counts[filepath.ToSlash(rel)] = n
			}
			return nil
		})
		if err != nil {
			t.Fatalf("扫描 %s 失败: %v", dir, err)
		}
	}
	return counts
}

func TestErrorLeakRatchet(t *testing.T) {
	measured := measuredErrorLeaks(t)

	// 扫描器自己坏掉（目录被挪走、正则失效）会让下面所有比较都「无违规」通过，
	// 所以先证明它确实扫到了存量。
	if len(measured) == 0 {
		t.Fatal("扫描器没有命中任何泄漏点：检查 errorLeakScanDirs 与 failHelperRE 是否失效")
	}

	if os.Getenv("ERROR_LEAK_RATCHET_DUMP") == "1" {
		keys := make([]string, 0, len(measured))
		for k := range measured {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		total := 0
		for _, k := range keys {
			t.Logf("\t\"%s\": %d,", k, measured[k])
			total += measured[k]
		}
		t.Logf("total=%d files=%d", total, len(keys))
		return
	}

	var growth, shrink []string
	for file, want := range errorLeakBaseline {
		got, ok := measured[file]
		switch {
		case !ok:
			shrink = append(shrink, file+" 已清零（基线 "+strconv.Itoa(want)+"）")
		case got > want:
			growth = append(growth, file+" "+strconv.Itoa(want)+"→"+strconv.Itoa(got))
		case got < want:
			shrink = append(shrink, file+" "+strconv.Itoa(want)+"→"+strconv.Itoa(got))
		}
	}
	for file, got := range measured {
		if _, ok := errorLeakBaseline[file]; !ok {
			growth = append(growth, file+" 新增泄漏点 "+strconv.Itoa(got))
		}
	}

	sort.Strings(growth)
	sort.Strings(shrink)
	if len(growth) > 0 {
		t.Errorf("响应里透出原始错误串的地方变多了，必须改用 common.FailWithErr/RespondError，而不是继续把 err.Error() 当公共消息:\n  %s",
			strings.Join(growth, "\n  "))
	}
	if len(shrink) > 0 {
		t.Errorf("泄漏面已下降，请同步把 errorLeakBaseline 改小并写进 CHANGELOG:\n  %s",
			strings.Join(shrink, "\n  "))
	}
}
