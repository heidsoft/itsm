# 后端覆盖率审计报告（Sprint v1.1）

> **生成时间**：2026-06-28
> **基线命令**：`go test ./service/... ./controller/... -coverprofile=cov.out -covermode=set`
> **范围**：itsm-backend（排除 ent 自动生成代码、migrations）

---

## 一、总体数字

| 维度 | 数值 | 备注 |
|:---|---:|:---|
| 已编译包总数 | **274** | 排除 ent/migrations 派生包后约 274 |
| service + controller 包覆盖率 | **13.6%** | 整体 5.2% 是因为其他 0 覆盖包被平均 |
| controller 包覆盖率 | **5.1%** | 单独看 controller 子包 |
| service 包数 | 121（含子目录）| — |
| **零覆盖包数**（手写代码）| **38** | 见下文清单 |
| 已覆盖包数 | 若干（bpmn、connector/marketplace 69.2%、rag 60%+）| — |

---

## 二、零覆盖包清单（按文件数排序 TOP 10）

| 排名 | 包路径 | 文件数 | 业务影响 | 优先级 |
|:---:|:---|---:|:---|:---:|
| 1 | `router/` | 7 | **入口路由注册** — 任何 API 上线都先经过此处 | 🔴 P0 |
| 2 | `service/approver/` | 6 | 审批人解析逻辑（与 BPMN 联动）| 🔴 P0 |
| 3 | `handlers/sla` | 5 | SLA 业务路由 | 🟠 P1 |
| 4 | `handlers/problem` | 5 | 问题管理路由 | 🟠 P1 |
| 5 | `handlers/knowledge` | 5 | 知识库路由 | 🟠 P1 |
| 6 | `handlers/incident` | 5 | **事件管理路由**（事件是核心 ITSM 能力） | 🔴 P0 |
| 7 | `handlers/service_request` | 4 | 服务请求路由 | 🟠 P1 |
| 8 | `repository/ticket/` | 3 | **工单数据访问层** | 🔴 P0 |
| 9 | `internal/bootstrap/` | 3 | 启动初始化 | 🟡 P2 |
| 10 | `pkg/eventbus/` | 2 | 事件总线 | 🟡 P2 |

> 完整 38 个零覆盖包清单见 `make coverage-audit`。

---

## 三、Sprint v1.1 补测优先级

按「**业务关键度 × 风险度 × 改动成本**」打分，优先级排序：

### 🔴 P0 — Sprint 立即补（本周）

1. **router 路由注册冒烟**
   - 文件：`router/router_test.go`
   - 目标：验证所有路由已注册，无 panic
   - 工作量：半天

2. **handlers/incident 路由测试**
   - 文件：`handlers/incident/handler_test.go`
   - 目标：HTTP handler 入口验证
   - 工作量：1 天

3. **repository/ticket CRUD 测试**
   - 文件：`repository/ticket/repository_test.go`
   - 目标：使用 sqlite in-memory 验证 CRUD
   - 工作量：1 天

### 🟠 P1 — 一个月内补

4. handlers/sla / problem / knowledge / service_request 路由测试
5. service/approver 单元测试（候选人组解析）
6. pkg/eventbus / pkg/cache / pkg/bpmn 工具函数测试
7. controller/auth / user / role 冒烟测试

### 🟡 P2 — 季度内补

8. internal/bootstrap 启动初始化测试
9. metrics / cache / config 等基础设施测试
10. domain/provisioning / cmd/* 工具脚本测试

---

## 四、推荐工具栈

| 场景 | 推荐 | 说明 |
|:---|:---|:---|
| Service 单测 | `testify/assert` + `testify/require` | 项目已使用 |
| DB 测试 | `enttest.NewClient()` + sqlite in-memory | 项目既有模式（见 [多租户隔离测试要求](多租户隔离测试要求.md)）|
| HTTP handler 测试 | `httptest` + `gin.TestMode` | 标准库 |
| Mock | 现有手写 mock 即可，引入 `mockery` v3 也可 | 看团队偏好 |
| 覆盖率 | `go test -coverprofile` + `go tool cover -html` | 内置工具链 |

---

## 五、阶段化覆盖率提升路径（与 ROADMAP.md 对齐）

| 阶段 | 时机 | 目标 | 关键动作 |
|:---|:---|:---:|:---|
| v1.0 GA | 已达成 | 1%（无回归 floor）| 仅防回退 |
| v1.1 (2026-Q3) | 当前 | **40%** | router/handlers/repository 路由+数据层补测 |
| v1.5 (2026-Q4) | 3 个月内 | **55%** | service/approver + eventbus + cache 工具层 |
| v2.0 (2027-Q2) | 6 个月内 | **70%** | controller 全部 + 集成测试 + 边界分支 |

> `backend-ci.yml` 会生成并上传后端覆盖率报告；当前不设置脱离实际基线的增量覆盖率硬门槛。

---

## 六、再生成审计快照

```bash
cd itsm-backend
go list ./... | grep -vE '^itsm-backend/(migrations|ent(/|$))' > /tmp/itsm_pkgs.txt
# 1) 零覆盖包清单
while read pkg; do
  short=${pkg#itsm-backend/}
  count=$(find "$short" -maxdepth 1 -name "*_test.go" -type f 2>/dev/null | wc -l)
  if [ "$count" = "0" ]; then
    src=$(find "$short" -maxdepth 1 -name "*.go" -not -name "*_test.go" -type f 2>/dev/null | wc -l)
    [ "$src" -gt 0 ] && printf "%-55s %d files\n" "$pkg" "$src"
  fi
done < /tmp/itsm_pkgs.txt | sort -k2 -rn

# 2) service + controller 综合覆盖率
go test ./service/... ./controller/... -count=1 -coverprofile=/tmp/cov.out -covermode=set 2>&1 | tail -3
go tool cover -func=/tmp/cov.out | tail -1
```

---

## 七、维护约定

- 任何 PR 触碰 service/* 或 controller/*，必须同步在 `*_test.go` 补至少 1 个用例。
- CI 通过 `backend-ci.yml` 持续生成覆盖率报告，并要求新增关键业务规则配套回归测试。
- 每次 Sprint 末重跑本审计，更新本文档的「零覆盖包清单」。
- 当零覆盖包数 < 10 时，将审计频率从「每周」降为「每月」。
