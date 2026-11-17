#!/usr/bin/env bash
# 一键注入 Email-to-Ticket 演示闭环数据（脱敏、可重复执行）。
#
# 依赖：本地 docker compose 起的 Postgres 容器（默认 itsm-postgres-prod）。
#       密码经容器自身环境变量传入 psql，绝不出现在本机命令行参数或日志里。
# 用法：bash scripts/seed-email-intake-demo.sh
#
# 环境变量（可选覆盖）：
#   COMPOSE_FILE    默认 docker-compose.prod.yml
#   ITSM_PG_CONTAINER 默认 itsm-postgres-prod
set -euo pipefail

# 通过容器名直连 Postgres（不依赖 compose 全量解析，避免必填变量缺失导致的插值报错）。
# 容器内自带 POSTGRES_PASSWORD / POSTGRES_USER / POSTGRES_DB，密码不外泄到本机。
PG_CONTAINER=${ITSM_PG_CONTAINER:-itsm-postgres-prod}

cat <<'SQL' | docker exec -i "$PG_CONTAINER" sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1'
-- 演示数据（demo 命名空间 id=990001，ON CONFLICT 幂等，可重复执行）
INSERT INTO service_customers (id, tenant_id, name, normalized_name, short_name, status, created_at, updated_at)
VALUES (990001, 1, '示例科技有限公司', '示例科技有限公司', '示例科技', 'active', now(), now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO support_contracts (id, tenant_id, customer_id, contract_number, normalized_contract_number, status, start_at, end_at, created_at, updated_at)
VALUES (990001, 1, 990001, 'SUP-2026-001', 'sup2026001', 'active', now(), now() + interval '1 year', now(), now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO email_conversations (id, tenant_id, conversation_token, status, canonical_data, confidence, version, last_message_at, created_at, updated_at, customer_id, support_contract_id)
VALUES (990001, 1, 'demo-seed-990001', 'MANUAL_REVIEW',
  '{"customerName":"示例科技有限公司","reportedContractNumber":"SUP-2026-001","title":"核心机房交换机端口故障","description":"核心机房接入交换机 GE0/1 端口频繁 up/down，导致对应网段业务间歇性中断，需现场排查。","impact":"high","urgency":"high","confidence":0.92}'::jsonb,
  0.92, 1, now(), now(), now(), 990001, 990001)
-- 幂等且自修复：已存在则刷新 canonical_data（避免旧演示数据残留导致确认失败）
ON CONFLICT (id) DO UPDATE SET
  canonical_data = EXCLUDED.canonical_data,
  confidence = EXCLUDED.confidence,
  version = EXCLUDED.version,
  status = EXCLUDED.status,
  customer_id = EXCLUDED.customer_id,
  support_contract_id = EXCLUDED.support_contract_id,
  updated_at = now();

INSERT INTO inbound_email_messages (id, tenant_id, provider, mailbox_instance_key, uid_validity, uid, from_address, to_addresses, subject, plain_text, raw_sha256, processing_status, conversation_id, received_at, created_at, updated_at)
VALUES (990001, 1, 'imap', 'demo-seed-990001', 990001, 990001, 'noc@example.com', '["helpdesk@example.com"]'::jsonb, '核心机房交换机端口故障',
  '核心机房接入交换机 GE0/1 端口频繁 up/down，导致对应网段业务间歇性中断。客户：示例科技有限公司，合同号：SUP-2026-001。',
  'demo-seed-990001-sha256', 'RECEIVED', 990001, now(), now(), now())
ON CONFLICT (id) DO NOTHING;
SQL

echo "✅ 演示数据已注入："
echo "   客户  示例科技有限公司  (service_customers id=990001)"
echo "   合同  SUP-2026-001     (support_contracts id=990001)"
echo "   会话  id=990001 (MANUAL_REVIEW, 置信度 0.92)"
echo ""
echo "下一步（确认即建单，需演示模式 env）："
echo "   1) 启动：docker compose -f docker-compose.prod.yml -f docker-compose.demo.yml --env-file .env.prod up -d itsm-backend itsm-worker"
echo "   2) 调用：POST /api/v1/email-intake/conversations/990001/confirm   body: {\"version\":1}"
echo "   3) 返回 INCIDENT_CREATED 即得到 INC-xxxx 事件。"
echo "   清理：bash scripts/unseed-email-intake-demo.sh"
