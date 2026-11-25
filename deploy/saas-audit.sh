#!/usr/bin/env bash
# SaaS 运维只读查询：谁访问了站点 / 谁登录了 demo。
# 用法（在 ECS 上 root 执行）：
#   ./saas-audit.sh visits [N]        # 最近的真实访问（nginx access.log，含公网 IP / UA / URI / 状态）
#   ./saas-audit.sh demo-login [N]    # demo 账号的登录审计（应用 audit_logs，含时间 / 结果 / status）
#   ./saas-audit.sh login-fails [N]   # 全站登录失败记录（暴力破解迹象）
#   ./saas-audit.sh top-ips [N]       # 访问来源 IP TOP N（快速识别异常来源）
#   ./saas-audit.sh blocked           # fail2ban 当前封禁概况
set -euo pipefail

ENV_FILE="${SAAS_ENV_FILE:-/opt/itsm-saas/.env.saas}"
ACCESS_LOG="${SAAS_ACCESS_LOG:-/var/log/nginx/access.log}"
CLOUDMESH_LOG="${SAAS_CLOUDMESH_LOG:-/var/log/nginx/cloudmesh_ssl_access.log}"
DEMO_UID="${SAAS_DEMO_UID:-2}"
N="${2:-20}"

die() { echo "ERROR: $*" >&2; exit 1; }

db_exec() {
  local u n p
  [ -f "$ENV_FILE" ] || die "找不到 $ENV_FILE"
  u=$(grep '^DB_USER=' "$ENV_FILE" | cut -d= -f2-)
  n=$(grep '^DB_NAME=' "$ENV_FILE" | cut -d= -f2-)
  p=$(grep '^DB_PASSWORD=' "$ENV_FILE" | cut -d= -f2-)
  docker exec -i -e PGPASSWORD="$p" itsm-saas-postgres psql -U "$u" -d "$n" -P pager=off
}

case "${1:-help}" in
  visits)
    echo "== 最近 $N 条 SaaS(itsm) 真实访问 [时间 客户端IP 状态 URI UA] =="
    tail -n "$N" "$ACCESS_LOG" 2>/dev/null | awk -F'"' '{print $1, "URI=" $2, "UA=" $6}' \
      | sed -E 's/  +/ /g' | tail -n "$N"
    echo
    echo "== 最近 $N 条 官网(cloudmesh) HTTPS 非静态访问（过滤 css/js/png）=="
    tail -n 300 "$CLOUDMESH_LOG" 2>/dev/null \
      | grep -vE '\.(css|js|png|ico|svg|woff2?|jpg|jpeg|webp)(\?| )' \
      | grep -vE '"(GET|HEAD) /_next/' | tail -n "$N"
    ;;
  demo-login)
    echo "== demo(uid=$DEMO_UID) 登录审计 [时间 结果 状态 IP] =="
    db_exec <<SQL
select to_char(created_at at time zone 'Asia/Shanghai','YYYY-MM-DD HH24:MI:SS') as bj_time,
       action, status_code, ip
from audit_logs
where user_id = $DEMO_UID and action in ('LOGIN_SUCCESS','LOGIN_FAILED')
order by created_at desc limit $N;
SQL
    ;;
  login-fails)
    echo "== 全站登录失败（暴力破解迹象）[北京时间 结果 状态 IP] =="
    db_exec <<SQL
select to_char(created_at at time zone 'Asia/Shanghai','YYYY-MM-DD HH24:MI:SS') as bj_time,
       action, status_code, ip, coalesce(user_id::text,'-') as uid
from audit_logs
where action = 'LOGIN_FAILED'
order by created_at desc limit $N;
SQL
    ;;
  top-ips)
    echo "== SaaS 访问来源 IP TOP ${N}（次数）=="
    awk '{print $1}' "$ACCESS_LOG" 2>/dev/null | sort | uniq -c | sort -rn | head -n "$N"
    ;;
  blocked)
    echo "== fail2ban 封禁概况 =="
    fail2ban-client status 2>/dev/null || { echo "fail2ban-client 不可用"; exit 0; }
    for jail in $(fail2ban-client status 2>/dev/null | sed -n 's/.*Jail list:[[:space:]]*//p' | tr ',' ' '); do
      echo "--- $jail ---"
      fail2ban-client status "$jail" 2>/dev/null | grep -E 'Currently banned|Total banned'
    done
    ;;
  help|*)
    cat <<'HELP'
用法: saas-audit.sh <命令> [N]
  visits [N]       最近 N 条真实访问（nginx access.log，公网 IP/UA/URI/状态）
  demo-login [N]   demo 账号登录审计（时间/结果/状态/IP）
  login-fails [N]  全站登录失败记录（暴力破解迹象）
  top-ips [N]      访问来源 IP TOP N
  blocked          fail2ban 封禁概况
说明: 全部只读；应用审计里的 ip 为容器内观察到的地址，公网真实来源以 visits 为准。
HELP
    ;;
esac
