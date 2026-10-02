#!/usr/bin/env bash
#
# e2e-isolated-stack.sh — 起一个一次性的 e2e 隔离栈，并在把测试环境变量交出去之前
# 用真实写路径证明「浏览器打的 :3001 代理」确实落在本栈的后端与数据库上。
#
# 为什么必须有这个脚本（2026-10-02 实测）：
#   next.config.ts 与 src/app/api/[...path]/route.ts 的代理 upstream 默认值都是
#   `http://localhost:8090`。本机 127.0.0.1:8090 由 `itsm-backend-prod`（DB_NAME=itsm_prod）
#   占着，[::1]:8090 由另一个会话起的宿主原生二进制占着。于是夹具的 POST /users、
#   工单写入会落到哪一个后端，只取决于 Node 解析 localhost 的顺序——「只打环回 :3000」
#   根本挡不住写路径。所以守卫不能只看请求 URL，必须证明代理链路指向本栈。
#
# 本脚本做的三件事：
#   1. 用新 project + 唯一容器名 + 独占端口（127.0.0.1:18090 / 127.0.0.1:3001）起栈，
#      postgres/redis 不发布宿主端口，前端 upstream 固定为 `http://backend:8090`；
#   2. canary 证明：经 :3001 代理登录 → 建一个随机名用户 → 用 docker exec 进本栈的
#      postgres 数出恰好 1 行。代理若指向别的后端，本栈库里就是 0 行，这里立刻红；
#   3. 把证明结果写成 0600 的 proof 文件（含口令，只落在临时目录），Playwright 侧的
#      isolatedBaseURL() 只认这个文件，从而把「跑前自动证明」变成硬门禁而不是文档约定。
#
# 用法：
#   scripts/e2e-isolated-stack.sh up                       # 构建+起栈+canary 证明
#   eval "$(scripts/e2e-isolated-stack.sh up --print-env)" # 同时把环境变量注入当前 shell
#   scripts/e2e-isolated-stack.sh status                   # 容器状态与 proof 摘要
#   scripts/e2e-isolated-stack.sh down                     # 拆栈并删除 proof
#
# 可调环境变量：
#   E2E_RUN_ID            运行号（默认取 PID），决定容器名/镜像 tag/project
#   E2E_ADMIN_PASSWORD    本栈 admin 口令；不传则随机生成，只写进 proof 文件
#   E2E_SKIP_BUILD=1      复用已构建的 itsm-e2e-backend:$E2E_RUN_ID（本地调试提速）
#   E2E_PROOF_DIR         proof 文件目录（默认 $TMPDIR/itsm-e2e-proofs）
#   E2E_UP_TIMEOUT        等待后端/前端就绪的秒数（默认 420，含 Next 冷编译）
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

say() { printf '[e2e-stack] %s\n' "$*" >&2; }
die() {
  say "FAIL: $*"
  exit 1
}
# 只在 canary/就绪阶段收集日志；收集动作本身绝不能让脚本二次失败。
dump_logs() {
  local dir
  dir="$(mktemp -d)"
  "${COMPOSE[@]}" logs --no-color --tail 60 >"$dir/compose.log" 2>&1 || true
  say "容器日志尾部（$dir/compose.log）："
  tail -30 "$dir/compose.log" 2>/dev/null >&2 || true
}
# macOS 上系统代理会劫持 localhost 请求（本机实测），所有探针一律显式绕过代理。
curl_loopback() { curl --noproxy '*' --silent --show-error --fail "$@"; }
# 前端首条请求会触发 Next 按需编译（本机 /login 实测 150s），5s 超时会误判未就绪。
http_code() { curl --noproxy '*' --silent --show-error -o /dev/null --max-time 30 -w '%{http_code}' "$1" 2>/dev/null || true; }
# 失败时把响应体原样打到 stderr：没有它就只能靠猜（本轮实测踩过 400 无 body 的情况）。
show_body() {
  local file="$1"
  [ -s "$file" ] || {
    say "响应体为空（$file）"
    return 0
  }
  say "响应体：$(head -c 400 "$file")"
}

ACTION="${1:-up}"
shift || true
PRINT_ENV=0
for arg in "$@"; do
  case "$arg" in
  --print-env) PRINT_ENV=1 ;;
  *) die "未知参数：$arg" ;;
  esac
done

E2E_RUN_ID="${E2E_RUN_ID:-$$}"
export E2E_RUN_ID # docker-compose.e2e.yml 靠它唯一化容器名与镜像 tag
PROJECT="${E2E_PROJECT:-itsm-e2e-${E2E_RUN_ID}}"
COMPOSE=(docker compose -p "$PROJECT" -f docker-compose.dev.yml -f docker-compose.e2e.yml)

# 测试基址必须是前端（走代理），不是后端裸端口：要证明的正是代理这一跳。
FRONTEND_URL="${E2E_FRONTEND_URL:-http://127.0.0.1:3001}"
BACKEND_URL="${E2E_BACKEND_URL:-http://127.0.0.1:18090}"
EXPECTED_PROXY_UPSTREAM="http://backend:8090"
BACKEND_CONTAINER="itsm-e2e-backend-${E2E_RUN_ID}"
FRONTEND_CONTAINER="itsm-e2e-frontend-${E2E_RUN_ID}"
PG_CONTAINER="itsm-e2e-postgres-${E2E_RUN_ID}"
PROOF_DIR="${E2E_PROOF_DIR:-${TMPDIR:-/tmp}/itsm-e2e-proofs}"
PROOF_FILE="$PROOF_DIR/${PROJECT}.json"
ENV_FILE="$PROOF_DIR/${PROJECT}.env"
PASSWORD_FILE="$PROOF_DIR/${PROJECT}.password"
UP_TIMEOUT="${E2E_UP_TIMEOUT:-420}"
PROOF_TTL_SECONDS="${E2E_PROOF_TTL_SECONDS:-21600}"

# ---------------------------------------------------------------------------
# down / status：只按 project 与 run id 精确操作，绝不碰 dev/prod 容器。
# ---------------------------------------------------------------------------
# compose 在「解析文件」阶段就会校验 itsm-init 的 ${E2E_ADMIN_PASSWORD:?}，所以拆栈/查状态
# 若不先给它一个值，整条命令直接失败（实测：down 报 required variable missing，而脚本
# 把 stderr 吞了，照样打印「已拆除」，容器与卷一个没少）。拆栈不创建任何容器，
# 因此这里用 up 时留存的口令文件，读不到就填占位值——占位值只用于让解析通过。
TEARDOWN_PLACEHOLDER='teardown-placeholder-not-a-real-password'
teardown_password_env() {
  [ -n "${E2E_ADMIN_PASSWORD:-}" ] && return 0
  if [ -f "$PASSWORD_FILE" ]; then
    E2E_ADMIN_PASSWORD="$(cat "$PASSWORD_FILE")"
  else
    E2E_ADMIN_PASSWORD="$TEARDOWN_PLACEHOLDER"
  fi
  export E2E_ADMIN_PASSWORD
}

if [ "$ACTION" = "down" ]; then
  command -v docker >/dev/null 2>&1 || die "docker 不可用"
  say "拆除 project=$PROJECT（含本栈卷）"
  teardown_password_env
  "${COMPOSE[@]}" down -v --remove-orphans >&2 || die "compose down 失败：见上面的 compose 输出"
  # 命令返回 0 不等于拆干净：按 project label 复核，残留就报出来，
  # 否则下一轮 up 会对着来源不明的容器做断言。
  leftover="$(docker ps -aq --filter "label=com.docker.compose.project=$PROJECT")"
  [ -z "$leftover" ] || {
    say "仍有容器属于 $PROJECT："
    docker ps -a --filter "label=com.docker.compose.project=$PROJECT" --format '  {{.Names}}\t{{.Status}}' >&2
    die "拆栈未生效，proof 文件保留以便重试"
  }
  rm -f "$PROOF_FILE" "$ENV_FILE" "$PASSWORD_FILE"
  say "已拆栈并删除 proof：$PROOF_FILE"
  exit 0
fi

if [ "$ACTION" = "status" ]; then
  teardown_password_env
  "${COMPOSE[@]}" ps
  if [ -f "$PROOF_FILE" ]; then
    say "proof=$PROOF_FILE"
    jq '{baseURL, proxyUpstream, imageID, canaryUser, expiresAtEpoch}' "$PROOF_FILE"
    if [ "$(date -u +%s)" -lt "$(jq -r '.expiresAtEpoch' "$PROOF_FILE")" ]; then
      say "proof 仍在有效期内"
    else
      say "proof 已过期：重新 up"
    fi
  else
    say "尚无 proof 文件（$PROOF_FILE）"
  fi
  exit 0
fi

[ "$ACTION" = "up" ] || die "未知动作：$ACTION（支持 up|down|status）"

# ---------------------------------------------------------------------------
# up：前置校验
# ---------------------------------------------------------------------------
command -v docker >/dev/null 2>&1 || die "docker 不可用"
docker info >/dev/null 2>&1 || die "docker daemon 未运行"
command -v jq >/dev/null 2>&1 || die "需要 jq（解析 canary 响应与写 proof）"

# 端口只能由本项目占用：本栈固定用 1870+ 这两个环回端口，若响应来自别的容器，
# 说明有来历不明的服务在同端口上，e2e 写请求就可能落到它那里。自己的残留则放行，
# 下面的 down -v 会重建同一 project。
OURS="$(docker ps -q --filter "label=com.docker.compose.project=$PROJECT" 2>/dev/null | tr -d '[:space:]')"
for probe in "$BACKEND_URL/api/v1/health" "$FRONTEND_URL/api/v1/csrf-token"; do
  if curl_loopback --max-time 3 -o /dev/null "$probe" 2>/dev/null && [ -z "$OURS" ]; then
    say "探测到 $probe 已有响应，但不属于 project=$PROJECT。一次性栈要求端口独占，请先："
    say "  E2E_RUN_ID=<当时的运行号> scripts/e2e-isolated-stack.sh down"
    die "拒绝与来历不明的服务共用 e2e 端口"
  fi
done

mkdir -p "$PROOF_DIR"
chmod 700 "$PROOF_DIR"

# 口令只来自环境或随机生成，仓库里不出现可用口令。
# 拆栈路径会填占位值（见 teardown_password_env），万一它泄漏到 up 就必须挡住，
# 否则等于用一个可猜测的口令起栈。
[ "${E2E_ADMIN_PASSWORD:-}" = "$TEARDOWN_PLACEHOLDER" ] &&
  die "E2E_ADMIN_PASSWORD 是拆栈占位值，不能用于起栈； unset 它或提供真实口令"
if [ -z "${E2E_ADMIN_PASSWORD:-}" ]; then
  E2E_ADMIN_PASSWORD="e2e-$(openssl rand -hex 12)"
  say "未提供 E2E_ADMIN_PASSWORD，已随机生成"
fi
# 口令会被拼进 JSON 请求体：字符集不合规时在这里挡住，别让 curl 报难懂的错。
printf '%s' "$E2E_ADMIN_PASSWORD" | grep -Eq '^[A-Za-z0-9_-]+$' \
  || die "E2E_ADMIN_PASSWORD 只能包含字母数字与 -_（canary 需要把它拼进 JSON 请求体）"
export E2E_ADMIN_PASSWORD
# 立刻落盘（0600）：canary 阶段失败时栈还在，没有这一份就只剩一个改不了口令的空壳容器。
printf '%s\n' "$E2E_ADMIN_PASSWORD" >"$PASSWORD_FILE"
chmod 600 "$PASSWORD_FILE"
say "admin 口令：$PASSWORD_FILE（一次性栈，勿提交）"

say "project=$PROJECT run=$E2E_RUN_ID frontend=$FRONTEND_URL backend(direct)=$BACKEND_URL"

# 先拆同名 project 的残留并清掉上一轮 canary 账号：postgres 卷在覆盖层里已 reset，
# 本栈每次都是空库，但同名容器/网络残留、以及被中断运行留下的 e2e-canary-* 用户
# 都会让 canary 计数失真（实测：上次失败留下的 canary 会撞用户名唯一约束）。
# 首轮没有残留可拆，非零退出属正常；但仍要把 compose 的输出留下来，
# 否则真·残留（上一轮被中断）会被静默跳过，canary 计数就不可归因。
if ! "${COMPOSE[@]}" down -v --remove-orphans 2>&1 | tail -5 >&2; then
  say "注意：同名 project 拆除未完全成功，后续按 project label 复核"
fi
if [ -n "$(docker ps -aq --filter "label=com.docker.compose.project=$PROJECT")" ]; then
  die "project=$PROJECT 仍有容器残留，compose down 未生效；先执行 scripts/e2e-isolated-stack.sh down"
fi
rm -f "$PROOF_FILE" "$ENV_FILE"

# ---------------------------------------------------------------------------
# 1. 构建被测后端镜像（itsm-init 与 backend 同一 Dockerfile，tag 复用即可）
# ---------------------------------------------------------------------------
if [ "${E2E_SKIP_BUILD:-0}" = "1" ]; then
  docker image inspect "itsm-e2e-backend:${E2E_RUN_ID}" >/dev/null 2>&1 \
    || die "E2E_SKIP_BUILD=1 但 itsm-e2e-backend:${E2E_RUN_ID} 不存在"
  say "复用已构建镜像 itsm-e2e-backend:${E2E_RUN_ID}"
else
  say "构建后端镜像（被测代码就编译在这里，禁止复用 dev/prod 镜像）"
  "${COMPOSE[@]}" build itsm-backend >/dev/null || {
    dump_logs
    die "后端镜像构建失败"
  }
fi
docker tag "itsm-e2e-backend:${E2E_RUN_ID}" "itsm-e2e-init:${E2E_RUN_ID}" >/dev/null
BUILT_IMAGE_ID="$(docker image inspect -f '{{.Id}}' "itsm-e2e-backend:${E2E_RUN_ID}")"
[ -n "$BUILT_IMAGE_ID" ] || die "取不到刚构建镜像的 ID"

# ---------------------------------------------------------------------------
# 2. 首装（空卷 migrate+seed）+ 起后端与前端
# ---------------------------------------------------------------------------
say "启动 postgres/redis 并等待健康"
"${COMPOSE[@]}" up -d --wait postgres redis >/dev/null 2>&1 || {
  dump_logs
  die "postgres/redis 未在健康时间内就绪"
}

say "执行 itsm-init（空卷首装：migrate + seed）"
"${COMPOSE[@]}" up --abort-on-container-exit --exit-code-from itsm-init itsm-init >/dev/null 2>&1 || {
  dump_logs
  die "itsm-init 首装失败——栈里没有基线数据，e2e 无从谈起"
}

say "启动 itsm-backend + itsm-frontend（--wait 直到依赖健康）"
"${COMPOSE[@]}" up -d --wait itsm-frontend >/dev/null 2>&1 || {
  dump_logs
  die "前端/后端未在健康时间内就绪"
}

# 跑起来的镜像必须就是刚构建的那个。compose 的 image 带 run id，正常必然成立，
# 但「改了代码却测到旧镜像」是这个仓库反复出现的假绿来源，所以显式断言。
RUNNING_IMAGE_ID="$(docker inspect -f '{{.Image}}' "$BACKEND_CONTAINER")"
[ "$RUNNING_IMAGE_ID" = "$BUILT_IMAGE_ID" ] ||
  die "运行中的后端镜像 $RUNNING_IMAGE_ID != 刚构建的 $BUILT_IMAGE_ID"
say "后端镜像一致：$BUILT_IMAGE_ID"

PROXY_UPSTREAM="$(docker exec "$FRONTEND_CONTAINER" printenv ITSM_BACKEND_URL 2>/dev/null || true)"
[ "$PROXY_UPSTREAM" = "$EXPECTED_PROXY_UPSTREAM" ] ||
  die "前端代理 upstream=$PROXY_UPSTREAM，期望 $EXPECTED_PROXY_UPSTREAM（绝不能是 localhost:8090）"

# ---------------------------------------------------------------------------
# 3. canary 证明：写路径必须穿过前端代理，并且只落在本栈数据库
# ---------------------------------------------------------------------------
wait_for() {
  local url="$1" label="$2" waited=0
  while [ "$waited" -lt "$UP_TIMEOUT" ]; do
    case "$(http_code "$url")" in
    2*)
      say "$label 就绪（$waited s）"
      return 0
      ;;
    esac
    sleep 5
    waited=$((waited + 5))
  done
  dump_logs
  die "$label 在 ${UP_TIMEOUT}s 内没有就绪：$url"
}
wait_for "$BACKEND_URL/api/v1/health" "后端(直连)"
# 前端第一条请求会触发 Next 按需编译，冷启动可能几分钟。
wait_for "$FRONTEND_URL/login" "前端(经代理)"

JAR="$(mktemp)"
CANARY="e2e-canary-$(openssl rand -hex 6)"
# service/user_service.go 的 validatePassword 要求同时含大写、小写、数字、特殊字符且 ≥12 位，
# 纯 hex 口令会被 400 挡掉（本轮实测踩过），所以口令模板固定带大小写与符号。
CANARY_PASSWORD="Canary#$(openssl rand -hex 8)7q"

csrf() {
  # CSRF 每次写操作后轮换，所以每次写之前都要重新取。
  curl -sS --noproxy '*' -b "$JAR" -c "$JAR" -H 'accept: application/json' \
    "$FRONTEND_URL/api/v1/csrf-token" | jq -r '.data.csrf_token // empty'
}

say "canary 1/3：经 $FRONTEND_URL 代理取 csrf-token"
TOKEN="$(csrf)"
[ -n "$TOKEN" ] || {
  dump_logs
  die "经代理取 csrf-token 失败：前端到后端的代理链路不通"
}

say "canary 2/3：经代理以 admin 登录"
LOGIN_STATUS="$(curl -sS --noproxy '*' -b "$JAR" -c "$JAR" -o /tmp/e2e-canary-login.json -w '%{http_code}' \
  --max-time 30 -X POST -H 'content-type: application/json' \
  -d "{\"username\":\"admin\",\"password\":\"$E2E_ADMIN_PASSWORD\"}" \
  "$FRONTEND_URL/api/v1/auth/login")"
[ "$LOGIN_STATUS" = "200" ] || {
  show_body /tmp/e2e-canary-login.json
  die "admin 经代理登录失败：HTTP $LOGIN_STATUS"
}
grep -q 'access_token' "$JAR" || die "cookie jar 里没有 access_token：会话没有经由代理建立"

say "canary 3/3：经代理创建一次性用户 $CANARY"
USER_STATUS="$(curl -sS --noproxy '*' -b "$JAR" -c "$JAR" -o /tmp/e2e-canary-user.json -w '%{http_code}' \
  --max-time 30 -X POST -H 'content-type: application/json' -H "X-CSRF-Token: $(csrf)" \
  -d "{\"username\":\"$CANARY\",\"email\":\"$CANARY@e2e.invalid\",\"name\":\"E2E Canary\",\"password\":\"$CANARY_PASSWORD\",\"role\":\"end_user\"}" \
  "$FRONTEND_URL/api/v1/users")"
[ "$USER_STATUS" = "200" ] || {
  show_body /tmp/e2e-canary-user.json
  say "排查提示：docker exec ${PG_CONTAINER} psql -U itsm_user -d itsm -c '\\d users'，或经直连端口复现：$BACKEND_URL/api/v1/users"
  die "经代理建用户失败：HTTP $USER_STATUS"
}

pg_sql() { docker exec "$PG_CONTAINER" psql -U itsm_user -d itsm -tAc "$1" | tr -d '[:space:]'; }
CANARY_ROWS="$(pg_sql "select count(*) from users where username='$CANARY'")"
[ "$CANARY_ROWS" = "1" ] || die "本栈库里 canary 用户有 $CANARY_ROWS 行（期望 1）：写请求经 $FRONTEND_URL 没有落到本栈，隔离不成立，禁止跑 e2e"
TOTAL_CANARY="$(pg_sql "select count(*) from users where username like 'e2e-canary-%'")"
[ "$TOTAL_CANARY" = "1" ] || die "canary 计数异常：库里已有 $TOTAL_CANARY 个 e2e-canary-* 用户"

# canary 只是探针，留着会干扰用户列表类断言；用真实 DELETE 路由清掉，失败不推翻证明。
CANARY_ID="$(pg_sql "select id from users where username='$CANARY'")"
DELETE_STATUS="$(curl -sS --noproxy '*' -b "$JAR" -c "$JAR" -o /dev/null -w '%{http_code}' \
  --max-time 30 -X DELETE -H "X-CSRF-Token: $(csrf)" "$FRONTEND_URL/api/v1/users/$CANARY_ID")"
[ "$DELETE_STATUS" = "200" ] || say "canary 用户清理未成功（HTTP $DELETE_STATUS），不影响 proof"

# ---------------------------------------------------------------------------
# 4. 写 proof：Playwright 侧的 isolatedBaseURL() 只认这个文件
# ---------------------------------------------------------------------------
NOW="$(date -u +%s)"
EXPIRES=$((NOW + PROOF_TTL_SECONDS))
ISO="$(date -u -r "$NOW" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u +%Y-%m-%dT%H:%M:%SZ)"
jq -n \
  --arg project "$PROJECT" \
  --arg runId "$E2E_RUN_ID" \
  --arg baseURL "$FRONTEND_URL" \
  --arg backendURL "$BACKEND_URL" \
  --arg proxyUpstream "$PROXY_UPSTREAM" \
  --arg imageID "$BUILT_IMAGE_ID" \
  --arg canary "$CANARY" \
  --arg adminPassword "$E2E_ADMIN_PASSWORD" \
  --arg createdAt "$ISO" \
  --argjson expiresAt "$EXPIRES" \
  '{project: $project, runId: $runId, baseURL: $baseURL, backendDirectURL: $backendURL,
    proxyUpstream: $proxyUpstream, imageID: $imageID, canaryUser: $canary,
    adminUsername: "admin", adminPassword: $adminPassword,
    createdAt: $createdAt, expiresAtEpoch: $expiresAt,
    proved: "POST /api/v1/users via frontend proxy landed in this project postgres"}' \
  >"$PROOF_FILE"
chmod 600 "$PROOF_FILE"

cat >"$ENV_FILE" <<EOF
# 由 scripts/e2e-isolated-stack.sh 生成，权限 0600，含一次性栈口令，勿提交。
export ITSM_E2E_ISOLATED_STACK=1
export ITSM_E2E_STACK_PROOF="$PROOF_FILE"
export PLAYWRIGHT_BASE_URL="$FRONTEND_URL"
export E2E_ADMIN_PASSWORD="$E2E_ADMIN_PASSWORD"
EOF
chmod 600 "$ENV_FILE"
rm -f "$JAR" /tmp/e2e-canary-login.json /tmp/e2e-canary-user.json

say "PASS: 代理→后端→数据库的写路径已证明；proof 有效期 $((PROOF_TTL_SECONDS / 3600)) 小时"
say "proof: $PROOF_FILE"
say "跑 e2e 前注入环境：set -a; . $ENV_FILE; set +a"

if [ "$PRINT_ENV" = "1" ]; then
  # 日志全走 stderr，stdout 只剩这四行，才能安全地 eval "$(… up --print-env)"。
  printf 'export ITSM_E2E_ISOLATED_STACK=1\nexport ITSM_E2E_STACK_PROOF=%q\nexport PLAYWRIGHT_BASE_URL=%q\nexport E2E_ADMIN_PASSWORD=%q\n' \
    "$PROOF_FILE" "$FRONTEND_URL" "$E2E_ADMIN_PASSWORD"
fi
