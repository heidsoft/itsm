#!/usr/bin/env bash
# ============================================================================
# 工作流跨域联调验收证据生成（workflow Pilot → GA 的人工签字前置）
# ============================================================================
#
# 用途：在**真实多租户环境**里自动收集可核对的联调证据，产出机器可读
# evidence JSON 与人类可读 markdown。它不代替验收人签字——签字栏始终留空。
#
# 默认只读：只做健康检查、两个租户登录、CSRF、能力矩阵、流程实例读面与
# 跨租户隔离探测，不会创建任何业务数据。
# 需要写链路证据时显式设置 ACCEPTANCE_ALLOW_WRITE=1，脚本会在目标环境创建
# 带 ACCEPTANCE- 前缀的工单/问题，并在报告里列出产生的 ID 供清理核对。
#
# 用法：
#   ITSM_BASE_URL=http://<host> \
#   TENANT_A_USER=... TENANT_A_PASS=... TENANT_A_CODE=... \
#   TENANT_B_USER=... TENANT_B_PASS=... TENANT_B_CODE=... \
#   bash scripts/acceptance/workflow-cross-domain-acceptance.sh
#
# 可选：ACCEPTANCE_ALLOW_WRITE=1、EVIDENCE_DIR=...、HEALTH_PATH=...、
#       ACCEPTANCE_TICKET_PAYLOAD='{"title":"...","..."}'、
#       ACCEPTANCE_PROBLEM_PAYLOAD='{...}'
#
# 退出码：0 = 无 FAIL（SKIP 允许，但报告会标出未覆盖项）；1 = 至少一项 FAIL。
# ============================================================================

set -uo pipefail

BASE_URL="${ITSM_BASE_URL:-http://localhost}"
API="${BASE_URL%/}/api/v1"
HEALTH_PATH="${HEALTH_PATH:-/health}"
OUT_DIR="${EVIDENCE_DIR:-output/acceptance}"
ALLOW_WRITE="${ACCEPTANCE_ALLOW_WRITE:-0}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
RUN_TAG="ACCEPTANCE-${STAMP}"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/itsm-acceptance.XXXXXX")"
JAR_A="${TMP_DIR}/tenant-a.cookies"
JAR_B="${TMP_DIR}/tenant-b.cookies"
RESULTS="${TMP_DIR}/results.jsonl"
mkdir -p "${OUT_DIR}"
JSON_OUT="${OUT_DIR}/workflow-cross-domain-${STAMP}.json"
MD_OUT="${OUT_DIR}/workflow-cross-domain-${STAMP}.md"

cleanup() { rm -rf "${TMP_DIR}"; }
trap cleanup EXIT

record() {
  local id="$1" desc="$2" status="$3" http_code="$4" detail="$5"
  python3 - "$RESULTS" "$id" "$desc" "$status" "$http_code" "$detail" <<'PY'
import json, sys
path, cid, desc, status, code, detail = sys.argv[1:]
with open(path, "a", encoding="utf-8") as f:
    f.write(json.dumps({
        "id": cid, "description": desc, "status": status,
        "httpStatus": code, "detail": detail,
    }, ensure_ascii=False) + "\n")
PY
  printf '%-34s %-6s HTTP %-4s %s\n' "$id" "$status" "$http_code" "$detail" | cut -c1-190
}

# request <method> <path> <jar> [body] [csrf-token] -> writes "$TMP_DIR/body", echoes status
request() {
  local method="$1" path="$2" jar="$3" body="${4:-}" token="${5:-}"
  local args=(-sS -m 15 -o "${TMP_DIR}/body" -w '%{http_code}'
              -X "$method" -b "$jar" -c "$jar" -H 'Content-Type: application/json')
  [ -n "$token" ] && args+=(-H "X-CSRF-Token: ${token}")
  [ -n "$body" ] && args+=(--data "$body")
  # shellcheck disable=SC2086
  curl "${args[@]}" "${BASE_URL%/}${path}" 2>"${TMP_DIR}/curl.err" || echo "000"
}

json_field() {
  python3 - "${TMP_DIR}/body" "$1" <<'PY' 2>/dev/null || true
import json, sys
d = json.load(open(sys.argv[1], encoding="utf-8"))
cur = d
for part in sys.argv[2].split("."):
    if isinstance(cur, dict):
        cur = cur.get(part)
    elif isinstance(cur, list) and part.isdigit():
        cur = cur[int(part)] if int(part) < len(cur) else None
    else:
        cur = None
if cur is None:
    print("")
elif isinstance(cur, (dict, list)):
    print(json.dumps(cur, ensure_ascii=False))
else:
    print(cur)
PY
}

csrf_token() {
  local jar="$1"
  request GET "/api/v1/csrf-token" "$jar" >/dev/null
  json_field "data.csrfToken"
}

login() {
  local user="$1" pass="$2" code="$3" jar="$4"
  local payload
  payload="$(python3 - "$user" "$pass" "$code" <<'PY'
import json, sys
print(json.dumps({"username": sys.argv[1], "password": sys.argv[2], "tenantCode": sys.argv[3]}))
PY
)"
  # 登录属于 public 组（CSRF 中间件挂在 auth 组），不带 token；
  # 登录成功后服务端刷新 token，后续每个写操作都必须重新取。
  request POST "/api/v1/auth/login" "$jar" "$payload"
}

echo "== 工作流跨域联调验收证据 =="
echo "BASE_URL=${BASE_URL}  MODE=$([ "$ALLOW_WRITE" = "1" ] && echo read-write || echo read-only)  TAG=${RUN_TAG}"
echo

# ---------- 1. 健康检查 ----------
code="$(request GET "${HEALTH_PATH}" "${JAR_A}")"
if [ "$code" = "200" ]; then
  record "health" "目标环境健康检查" "PASS" "$code" "$(head -c 120 "${TMP_DIR}/body")"
else
  record "health" "目标环境健康检查" "FAIL" "$code" "后端不可达，后续检查无意义"
fi

# ---------- 2. 两个真实租户登录（httpOnly cookie jar） ----------
for side in A B; do
  u="TENANT_${side}_USER"; p="TENANT_${side}_PASS"; c="TENANT_${side}_CODE"
  user="${!u:-}"; pass="${!p:-}"; tcode="${!c:-}"; jar="${TMP_DIR}/tenant-$([ "$side" = A ] && echo a || echo b).cookies"
  if [ -z "$user" ] || [ -z "$pass" ]; then
    record "login_${side}" "租户 ${side} 登录" "SKIP" "-" "未配置 ${u}/${p}"
    continue
  fi
  code="$(login "$user" "$pass" "$tcode" "$jar")"
  if [ "$code" = "200" ] && [ -s "$jar" ]; then
    record "login_${side}" "租户 ${side} 登录" "PASS" "$code" "httpOnly cookie 已写入 jar（响应体不含 token）"
  else
    record "login_${side}" "租户 ${side} 登录" "FAIL" "$code" "$(head -c 160 "${TMP_DIR}/body")"
  fi
done

# ---------- 3. CSRF 契约 ----------
for side in A B; do
  jar="${TMP_DIR}/tenant-$([ "$side" = A ] && echo a || echo b).cookies"
  token="$(csrf_token "$jar")"
  if [ -n "$token" ]; then
    record "csrf_${side}" "租户 ${side} CSRF 凭据下发" "PASS" "200" "data.csrfToken 已取得，写操作将带 X-CSRF-Token"
  else
    record "csrf_${side}" "租户 ${side} CSRF 凭据下发" "FAIL" "-" "未解析到 data.csrfToken"
  fi
done

# ---------- 4. 能力矩阵 workflow 口径 ----------
code="$(request GET "/api/v1/capabilities" "$JAR_A")"
caps="$(cat "${TMP_DIR}/body" 2>/dev/null || true)"
workflow_line="$(python3 - "$caps" <<'PY' 2>/dev/null || true
import json, sys
try:
    d = json.loads(sys.argv[1])
except Exception:
    print(""); raise SystemExit
items = d.get("data") or d.get("capabilities") or []
if isinstance(items, dict):
    items = items.get("items") or items.get("capabilities") or []
for it in items if isinstance(items, list) else []:
    if isinstance(it, dict) and it.get("key") == "workflow":
        print(json.dumps({"maturity": it.get("maturity"), "pilotReason": it.get("pilotReason")}, ensure_ascii=False))
        break
PY
)"
if [ "$code" = "200" ] && [ -n "$workflow_line" ]; then
  record "capability_workflow" "能力矩阵 workflow 成熟度口径" "PASS" "$code" "$workflow_line"
else
  record "capability_workflow" "能力矩阵 workflow 成熟度口径" "FAIL" "$code" "未读到 workflow 条目：$(printf '%s' "$caps" | head -c 160)"
fi

# ---------- 5. 两租户流程实例读面 + 跨租户隔离 ----------
instance_id_a=""; instance_id_b=""
for side in A B; do
  jar="${TMP_DIR}/tenant-$([ "$side" = A ] && echo a || echo b).cookies"
  code="$(request GET "/api/v1/bpmn/process-instances?page=1&pageSize=5" "$jar")"
  parsed="$(python3 - "${TMP_DIR}/body" <<'PY' 2>/dev/null || true
import json, sys
try:
    d = json.load(open(sys.argv[1], encoding="utf-8"))
except Exception:
    print("|parse-error"); raise SystemExit
data = d.get("data", d)
rows = data.get("items") or data.get("instances") or data.get("list") or []
if not isinstance(rows, list):
    rows = []
total = data.get("total", len(rows))
first = ""
if rows and isinstance(rows[0], dict):
    first = rows[0].get("id") or rows[0].get("processInstanceId") or ""
print(f"{total}|{first}")
PY
)"
  total="${parsed%%|*}"; first="${parsed##*|}"
  if [ "$code" = "200" ] && [ -n "$total" ] && [ "$total" != "parse-error" ]; then
    record "instances_${side}" "租户 ${side} 流程实例读面" "PASS" "$code" "可见实例数=${total}，首条 id=${first:-none}"
    [ "$side" = A ] && instance_id_a="$first" || instance_id_b="$first"
  else
    record "instances_${side}" "租户 ${side} 流程实例读面" "FAIL" "$code" "$(head -c 160 "${TMP_DIR}/body")"
  fi
done

if [ -n "$instance_id_b" ]; then
  code="$(request GET "/api/v1/bpmn/process-instances/${instance_id_b}" "$JAR_A")"
  case "$code" in
    403|404) record "cross_tenant_isolation" "A 会话读取 B 的流程实例" "PASS" "$code" "跨租户按不存在/无权限拒绝" ;;
    200)     record "cross_tenant_isolation" "A 会话读取 B 的流程实例" "FAIL" "$code" "租户隔离被击穿" ;;
    *)       record "cross_tenant_isolation" "A 会话读取 B 的流程实例" "FAIL" "$code" "非预期状态码 $(head -c 120 "${TMP_DIR}/body")" ;;
  esac
else
  record "cross_tenant_isolation" "A 会话读取 B 的流程实例" "SKIP" "-" "租户 B 无可读实例，需在写模式下先创建"
fi

# ---------- 6. 写链路（需显式授权） ----------
if [ "$ALLOW_WRITE" = "1" ]; then
  ticket_payload="${ACCEPTANCE_TICKET_PAYLOAD:-}"
  problem_payload="${ACCEPTANCE_PROBLEM_PAYLOAD:-}"
  if [ -z "$ticket_payload" ] || [ -z "$problem_payload" ]; then
    record "write_chain" "跨业务域创建与流程推进" "SKIP" "-" "需同时提供 ACCEPTANCE_TICKET_PAYLOAD / ACCEPTANCE_PROBLEM_PAYLOAD"
  else
    token="$(csrf_token "$JAR_A")"
    code="$(request POST "/api/v1/tickets" "$JAR_A" "$ticket_payload" "$token")"
    ticket_id="$(json_field "data.id")"
    if [ "$code" = "200" ] || [ "$code" = "201" ]; then
      record "create_ticket_a" "租户 A 创建工单（工单域）" "PASS" "$code" "ticketId=${ticket_id:-unknown} tag=${RUN_TAG}"
      sleep 2   # 服务请求经 outbox 异步起流
      code="$(request GET "/api/v1/tickets/${ticket_id}/workflow/state-v2" "$JAR_A")"
      actions="$(json_field "data.availableActions")"
      if [ "$code" = "200" ] && [ -n "$actions" ] && [ "$actions" != "[]" ]; then
        record "state_v2_a" "state-v2 可用动作" "PASS" "$code" "availableActions=${actions}"
      else
        record "state_v2_a" "state-v2 可用动作" "FAIL" "$code" "$(head -c 160 "${TMP_DIR}/body")"
      fi
      code="$(request GET "/api/v1/bpmn/process-instances?page=1&pageSize=20" "$JAR_A")"
      if [ "$code" = "200" ]; then
        record "workflow_start_a" "工单创建触发 workflow.start" "SKIP" "$code" "实例列表可读；businessKey=ticket:${ticket_id} 的实例存在性需人工核对后补证据"
      else
        record "workflow_start_a" "工单创建触发 workflow.start" "FAIL" "$code" "$(head -c 160 "${TMP_DIR}/body")"
      fi
      token="$(csrf_token "$JAR_A")"
      code="$(request POST "/api/v1/tickets/workflow/accept" "$JAR_A" "{\"ticketId\":${ticket_id}}" "$token")"
      if [ "$code" = "200" ]; then
        record "accept_ticket_a" "工单接单动作与 BPMN 待办同事务推进" "PASS" "$code" "已提交，需按实例 currentActivityId 变化复核"
      else
        record "accept_ticket_a" "工单接单动作与 BPMN 待办同事务推进" "FAIL" "$code" "$(head -c 160 "${TMP_DIR}/body")"
      fi
    else
      record "create_ticket_a" "租户 A 创建工单（工单域）" "FAIL" "$code" "$(head -c 160 "${TMP_DIR}/body")"
    fi

    token="$(csrf_token "$JAR_B")"
    code="$(request POST "/api/v1/problems" "$JAR_B" "$problem_payload" "$token")"
    if [ "$code" = "200" ] || [ "$code" = "201" ]; then
      prob_id="$(json_field "data.id")"
      record "create_problem_b" "租户 B 创建问题（问题域）" "PASS" "$code" "problemId=${prob_id:-unknown} tag=${RUN_TAG}"
      sleep 2
      record "workflow_start_b" "问题创建触发 workflow.start" "SKIP" "-" "按 businessKey=problem:${prob_id} 在实例列表核对后补证据"
    else
      record "create_problem_b" "租户 B 创建问题（问题域）" "FAIL" "$code" "$(head -c 160 "${TMP_DIR}/body")"
    fi
  fi
  record "cli_compensation_dryrun" "补偿显式触发 dry-run 证据" "SKIP" "-" "由运维执行：cd itsm-backend && go run ./cmd/bpmn-compensation -tenant-id <A> -instance-id <id> -actor-id <id> -reason acceptance"
  record "cli_migration_dryrun" "存量待办迁移 dry-run 证据" "SKIP" "-" "由运维执行：cd itsm-backend && go run ./cmd/pending-task-migration -tenant-id <A>"
else
  record "write_chain" "跨业务域创建与流程推进" "SKIP" "-" "只读模式；设 ACCEPTANCE_ALLOW_WRITE=1 并给 payload 才执行"
  record "cli_compensation_dryrun" "补偿显式触发 dry-run 证据" "SKIP" "-" "只读模式未执行"
  record "cli_migration_dryrun" "存量待办迁移 dry-run 证据" "SKIP" "-" "只读模式未执行"
fi

# ---------- 7. 汇总与签字栏（永远留空，不由脚本代签） ----------
python3 - "$RESULTS" "$JSON_OUT" "$MD_OUT" "$BASE_URL" "$ALLOW_WRITE" "$RUN_TAG" <<'PY'
import json, sys, datetime
results_path, json_out, md_out, base_url, allow_write, run_tag = sys.argv[1:]
checks = []
with open(results_path, encoding="utf-8") as f:
    for line in f:
        line = line.strip()
        if line:
            checks.append(json.loads(line))
summary = {s: sum(1 for c in checks if c["status"] == s) for s in ("PASS", "FAIL", "SKIP")}
payload = {
    "generatedAt": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    "baseUrl": base_url,
    "mode": "read-write" if allow_write == "1" else "read-only",
    "runTag": run_tag,
    "summary": summary,
    "checks": checks,
    "signoff": {
        "required": True,
        "signedBy": None,
        "signedAt": None,
        "conclusion": None,
        "note": "人工验收签字不可由脚本或 AI 代签；签字栏必须留空直至验收人填写。",
    },
}
with open(json_out, "w", encoding="utf-8") as f:
    json.dump(payload, f, ensure_ascii=False, indent=2)

lines = [
    "# 工作流跨域联调验收证据",
    "",
    f"- 生成时间：{payload['generatedAt']}",
    f"- 目标环境：`{base_url}`",
    f"- 执行模式：{payload['mode']}（runTag `{run_tag}`）",
    f"- 结果：PASS {summary['PASS']} / FAIL {summary['FAIL']} / SKIP {summary['SKIP']}",
    "",
    "## 检查项",
    "",
    "| ID | 检查内容 | 结果 | HTTP | 证据 |",
    "| --- | --- | --- | --- | --- |",
]
for c in checks:
    detail = str(c["detail"]).replace("|", "\\|").replace("\n", " ")[:220]
    lines.append(f"| `{c['id']}` | {c['description']} | {c['status']} | {c['httpStatus']} | {detail} |")
lines += [
    "",
    "## 自动化证据边界",
    "",
    "本脚本只证明「目标环境上这些读写面可达且租户隔离成立」。以下事项**不在**自动化范围内，",
    "必须由验收人按 `docs/acceptance/workflow-cross-domain-signoff.md` 逐项确认并贴证据：",
    "",
    "1. 工单/问题/变更等各业务域在同一真实租户内的完整生命周期走通（创建 → 起流 → 动作 → 终态）。",
    "2. ServiceTask 副作用完成凭据（`_serviceTaskReceipts`）与流程推进在同一提交里可见，崩溃重放不重复发通知。",
    "3. 补偿显式触发（dry-run → apply）在真实实例上按逆序回滚，重复触发幂等。",
    "4. 存量 pending 待办迁移（dry-run → apply）在真实租户上可回滚、可核对审计。",
    "5. 跨租户零串写、零审计泄漏，且所有 SKIP 项已补做。",
    "",
    "## 人工验收签字（留空，待验收人填写）",
    "",
    "| 项目 | 填写 |",
    "| --- | --- |",
    "| 验收人 |  |",
    "| 验收日期 |  |",
    "| 验收环境 / 版本 |  |",
    "| 结论（通过 / 不通过 / 有条件通过） |  |",
    "| 遗留问题与跟进批次 |  |",
    "| 签字 |  |",
    "",
    "> 签字栏为空即表示**尚未验收**；能力矩阵 workflow 必须保持 Pilot，不得据此晋升 GA。",
    "",
]
with open(md_out, "w", encoding="utf-8") as f:
    f.write("\n".join(lines))
print(f"SUMMARY PASS={summary['PASS']} FAIL={summary['FAIL']} SKIP={summary['SKIP']}")
PY

echo
echo "证据文件："
echo "  ${JSON_OUT}"
echo "  ${MD_OUT}"
if grep -q '"status": "FAIL"' "$RESULTS"; then
  echo "结论：存在 FAIL，本次不得作为验收通过证据。"
  exit 1
fi
echo "结论：无 FAIL；SKIP 项需按签字清单补做后方可签字。"
exit 0
