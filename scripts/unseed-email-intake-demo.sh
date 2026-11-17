#!/usr/bin/env bash
# 一键清除 Email-to-Ticket 演示数据（含可能已建的演示事件，FK 安全序，原子事务）。
#
# 依赖：本地 docker compose 起的 Postgres 容器（默认 itsm-postgres-prod）。
#       密码经容器自身环境变量传入 psql，绝不出现在命令行参数或日志里。
# 用法：bash scripts/unseed-email-intake-demo.sh
#
# 环境变量（可选覆盖）：
#   COMPOSE_FILE      默认 docker-compose.prod.yml
#   ITSM_PG_CONTAINER 默认 itsm-postgres-prod
set -euo pipefail

# 通过容器名直连 Postgres（不依赖 compose 全量解析，避免必填变量缺失导致的插值报错）。
# 容器内自带 POSTGRES_PASSWORD / POSTGRES_USER / POSTGRES_DB，密码不外泄到本机。
PG_CONTAINER=${ITSM_PG_CONTAINER:-itsm-postgres-prod}

cat <<'SQL' | docker exec -i "$PG_CONTAINER" sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1'
BEGIN;
  -- 先清演示事件及其全部子表（按客户/合同归属定位，避免误伤真实数据）
  DELETE FROM incident_alerts WHERE incident_id IN (SELECT id FROM incidents WHERE email_conversation_id=990001);
  DELETE FROM incident_events WHERE incident_id IN (SELECT id FROM incidents WHERE email_conversation_id=990001);
  DELETE FROM incident_metrics WHERE incident_id IN (SELECT id FROM incidents WHERE email_conversation_id=990001);
  DELETE FROM configuration_item_incidents WHERE incident_id IN (SELECT id FROM incidents WHERE email_conversation_id=990001);
  DELETE FROM incident_related_incidents WHERE incident_id IN (SELECT id FROM incidents WHERE email_conversation_id=990001) OR parent_incident_id IN (SELECT id FROM incidents WHERE email_conversation_id=990001);
  DELETE FROM problem_incidents WHERE incident_id IN (SELECT id FROM incidents WHERE email_conversation_id=990001);
  DELETE FROM incidents WHERE email_conversation_id=990001;
  -- 再清邮件链路
  DELETE FROM inbound_email_messages WHERE conversation_id=990001;
  DELETE FROM email_outbound_messages WHERE conversation_id=990001;
  DELETE FROM email_conversations WHERE id=990001;
  -- 最后清合同与客户
  DELETE FROM external_contract_references WHERE support_contract_id=990001;
  DELETE FROM support_contracts WHERE id=990001;
  DELETE FROM customer_branches WHERE customer_id=990001;
  DELETE FROM service_customers WHERE id=990001;
COMMIT;
SQL

echo "✅ 演示数据已清除（含 demo 事件，若已建）。prod 恢复空 NOC 队列状态。"
