-- =====================================================================
-- ITSM Initial Seed: SLA Policies + Knowledge SOPs + CMDB CIs/Relations
-- =====================================================================
--
-- PURPOSE
--   为新租户/现有租户一次性补齐运维运营所必需的 P0/T0 基础模板：
--     * SLA 策略矩阵（覆盖 P0-P3 + 服务请求/变更/问题子类型）
--     * 12 篇 ITIL/ITSM SOP 知识文章（事件、变更、问题、请求、CMDB、KB、SLA）
--     * 16 个核心 CI 实例（覆盖所有 8 类 CI 类型）
--     * 15 条 CI 关系（覆盖典型业务拓扑，含关键依赖）
--
-- 所属数据分类（参考 production-data-initialization-blueprint）：
--   - sla_policies     : T0 租户模板（可被租户覆盖）
--   - knowledge_articles: D0 知识模板（可由租户编辑/扩展）
--   - configuration_items / ci_relationships : R0 拓扑基线
--     （CI 关系属于拓扑资产，不是业务单据；初始化阶段放置是为了
--      把 topology_invariant 等测试落到稳定资产上）
--
-- DESIGN
--   - 完全幂等（idempotent），可重入：
--       * sla_policies      : 通过 (tenant_id, name) 自然键 SELECT-then-INSERT/UPDATE
--       * knowledge_articles : 通过 (tenant_id, title) 自然键 SELECT-then-INSERT/UPDATE
--       * configuration_items: 通过唯一索引 (tenant_id, serial_number) ON CONFLICT
--       * ci_relationships   : 通过唯一索引 (source_ci_id, target_ci_id, relationship_type)
--   - 写入目标租户 (tenant_id = 1)，不会出现跨租户污染。
--   - RLS 兼容：开头用 SET LOCAL app.current_tenant_id = 1 满足 sla_policies
--     上的 tenant_isolation 策略（其它三张表当前未启用 RLS，但保持兼容）。
--   - 单事务包裹，确保 CI 实例 + 关系原子提交（任一失败 → 全部回滚）。
--
-- INVOCATION
--   PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -U $DB_USER -d $DB_NAME \
--       -v ON_ERROR_STOP=1 -f scripts/seed_initial_kb_cmdb_sla.sql
--
--   docker exec -i itsm-postgres-dev psql -U itsm_user -d itsm \
--       -v ON_ERROR_STOP=1 < scripts/seed_initial_kb_cmdb_sla.sql
--
-- AUTHOR
--   平台团队（managed by itsm-backend/scripts），任何修改请同步更新
--   plans/production-data-initialization-blueprint.md 第 4.3 节新租户开通模板。

\set ON_ERROR_STOP on
\set QUIET on

BEGIN;

-- ---------------------------------------------------------------------
-- 0) 上下文：租户 + 作者用户 + 当前时间
-- ---------------------------------------------------------------------
SET LOCAL app.current_tenant_id = 1;

DO $$
DECLARE
  v_tenant_id bigint := 1;
  v_author_id bigint;
BEGIN
  SELECT id INTO v_author_id FROM users
   WHERE username = 'admin' AND tenant_id = v_tenant_id
   LIMIT 1;
  IF v_author_id IS NULL THEN
    RAISE EXCEPTION 'seed_initial: admin user (tenant_id=1) not found, abort';
  END IF;

  CREATE TEMP TABLE _seed_ctx ON COMMIT DROP AS
    SELECT v_tenant_id::bigint AS tenant_id, v_author_id::bigint AS author_id;
END $$;

-- ---------------------------------------------------------------------
-- 1) SLA 策略（7 条：P0-P3 + 服务请求 + 变更 + 问题）
-- ---------------------------------------------------------------------
-- 注：现有表中已存在 3 条 (P1/P2/P3)，本块更新其字段并补足缺失 4 条。
-- 自然键 (tenant_id, name) 不是唯一索引，故走 SELECT-then-INSERT/UPDATE。

DO $$
DECLARE
  v_tenant_id bigint := 1;
  v_id bigint;
  -- P0-致命响应
  v_bh jsonb := jsonb_build_object(
    'work_days', jsonb_build_array(1,2,3,4,5,6,7),
    'start_time','00:00','end_time','23:59','time_zone','Asia/Shanghai',
    'holiday_list', jsonb_build_array()
  );
  v_es jsonb := jsonb_build_object('thresholds', jsonb_build_array(
    jsonb_build_object('percent', 50, 'notify', jsonb_build_array('ops_manager','ops_director')),
    jsonb_build_object('percent', 80, 'notify', jsonb_build_array('it_director','ops_director'), 'page', true),
    jsonb_build_object('percent', 100,'notify', jsonb_build_array('dept_manager'), 'page', true)
  ));
BEGIN
  -- P0-致命响应
  SELECT id INTO v_id FROM sla_policies WHERE tenant_id=v_tenant_id AND name='P0-致命响应';
  IF v_id IS NULL THEN
    INSERT INTO sla_policies(name,description,customer_tier,ticket_type,priority,
      response_time_minutes,resolution_time_minutes,business_hours,
      exclude_weekends,exclude_holidays,escalation_rules,
      is_active,priority_score,tenant_id,created_at,updated_at)
    VALUES ('P0-致命响应',
      'P0/致命级事件响应——业务整体不可用，触发立即响应与跨团队升级。',
      'platinum','incident','critical',5,60, v_bh, false,false, v_es,
      true, 100, v_tenant_id, NOW(), NOW());
  ELSE
    UPDATE sla_policies SET description='P0/致命级事件响应——业务整体不可用，触发立即响应与跨团队升级。',
      customer_tier='platinum', ticket_type='incident', priority='critical',
      response_time_minutes=5, resolution_time_minutes=60,
      business_hours=jsonb_build_object(
        'work_days', jsonb_build_array(1,2,3,4,5,6,7),
        'start_time','00:00','end_time','23:59','time_zone','Asia/Shanghai',
        'holiday_list', jsonb_build_array()
      ),
      exclude_weekends=false, exclude_holidays=false,
      escalation_rules=jsonb_build_object('thresholds', jsonb_build_array(
        jsonb_build_object('percent', 50, 'notify', jsonb_build_array('ops_manager','ops_director')),
        jsonb_build_object('percent', 80, 'notify', jsonb_build_array('it_director','ops_director'), 'page', true),
        jsonb_build_object('percent', 100,'notify', jsonb_build_array('dept_manager'), 'page', true)
      )),
      is_active=true, priority_score=100,
      updated_at=NOW()
    WHERE id=v_id;
  END IF;

  -- P1-紧急响应
  SELECT id INTO v_id FROM sla_policies WHERE tenant_id=v_tenant_id AND name='P1-紧急响应';
  IF v_id IS NULL THEN
    INSERT INTO sla_policies(name,description,customer_tier,ticket_type,priority,
      response_time_minutes,resolution_time_minutes,business_hours,
      exclude_weekends,exclude_holidays,escalation_rules,
      is_active,priority_score,tenant_id,created_at,updated_at)
    VALUES ('P1-紧急响应',
      'P1/高级事件响应——核心业务功能受限，1 小时内需要稳定处理路径。',
      'gold','incident','high',15,240,
      jsonb_build_object('work_days', jsonb_build_array(1,2,3,4,5,6,7),
                         'start_time','00:00','end_time','23:59','time_zone','Asia/Shanghai'),
      false,false,
      jsonb_build_object('thresholds', jsonb_build_array(
        jsonb_build_object('percent', 50, 'notify', jsonb_build_array('ops_manager')),
        jsonb_build_object('percent', 80, 'notify', jsonb_build_array('ops_director','it_director'))
      )),
      true, 90, v_tenant_id, NOW(), NOW());
  ELSE
    UPDATE sla_policies SET
      description='P1/高级事件响应——核心业务功能受限，1 小时内需要稳定处理路径。',
      customer_tier='gold', ticket_type='incident', priority='high',
      response_time_minutes=15, resolution_time_minutes=240,
      business_hours=jsonb_build_object('work_days', jsonb_build_array(1,2,3,4,5,6,7),
                       'start_time','00:00','end_time','23:59','time_zone','Asia/Shanghai'),
      exclude_weekends=false, exclude_holidays=false,
      escalation_rules=jsonb_build_object('thresholds', jsonb_build_array(
        jsonb_build_object('percent', 50, 'notify', jsonb_build_array('ops_manager')),
        jsonb_build_object('percent', 80, 'notify', jsonb_build_array('ops_director','it_director'))
      )),
      is_active=true, priority_score=90,
      updated_at=NOW()
    WHERE id=v_id;
  END IF;

  -- P2-标准响应
  SELECT id INTO v_id FROM sla_policies WHERE tenant_id=v_tenant_id AND name='P2-标准响应';
  IF v_id IS NULL THEN
    INSERT INTO sla_policies(name,description,customer_tier,ticket_type,priority,
      response_time_minutes,resolution_time_minutes,business_hours,
      exclude_weekends,exclude_holidays,escalation_rules,
      is_active,priority_score,tenant_id,created_at,updated_at)
    VALUES ('P2-标准响应',
      'P2/中级事件响应——常规业务影响，业务时间 8 小时内解决。',
      'silver','incident','medium',30,480,
      jsonb_build_object('work_days', jsonb_build_array(1,2,3,4,5),
                         'start_time','09:00','end_time','18:00','time_zone','Asia/Shanghai'),
      true,true,
      jsonb_build_object('thresholds', jsonb_build_array(
        jsonb_build_object('percent', 80, 'notify', jsonb_build_array('ops_manager'))
      )),
      true, 60, v_tenant_id, NOW(), NOW());
  ELSE
    UPDATE sla_policies SET
      description='P2/中级事件响应——常规业务影响，业务时间 8 小时内解决。',
      customer_tier='silver', ticket_type='incident', priority='medium',
      response_time_minutes=30, resolution_time_minutes=480,
      business_hours=jsonb_build_object('work_days', jsonb_build_array(1,2,3,4,5),
                       'start_time','09:00','end_time','18:00','time_zone','Asia/Shanghai'),
      exclude_weekends=true, exclude_holidays=true,
      escalation_rules=jsonb_build_object('thresholds', jsonb_build_array(
        jsonb_build_object('percent', 80, 'notify', jsonb_build_array('ops_manager'))
      )),
      is_active=true, priority_score=60,
      updated_at=NOW()
    WHERE id=v_id;
  END IF;

  -- P3-一般响应
  SELECT id INTO v_id FROM sla_policies WHERE tenant_id=v_tenant_id AND name='P3-一般响应';
  IF v_id IS NULL THEN
    INSERT INTO sla_policies(name,description,customer_tier,ticket_type,priority,
      response_time_minutes,resolution_time_minutes,business_hours,
      exclude_weekends,exclude_holidays,escalation_rules,
      is_active,priority_score,tenant_id,created_at,updated_at)
    VALUES ('P3-一般响应',
      'P3/低优先级事件或服务咨询，可延后到次日业务时间处理。',
      'bronze','incident','low',120,1440,
      jsonb_build_object('work_days', jsonb_build_array(1,2,3,4,5),
                         'start_time','09:00','end_time','18:00','time_zone','Asia/Shanghai'),
      true,true,
      jsonb_build_object('thresholds', jsonb_build_array()),
      true, 30, v_tenant_id, NOW(), NOW());
  ELSE
    UPDATE sla_policies SET
      description='P3/低优先级事件或服务咨询，可延后到次日业务时间处理。',
      customer_tier='bronze', ticket_type='incident', priority='low',
      response_time_minutes=120, resolution_time_minutes=1440,
      business_hours=jsonb_build_object('work_days', jsonb_build_array(1,2,3,4,5),
                       'start_time','09:00','end_time','18:00','time_zone','Asia/Shanghai'),
      exclude_weekends=true, exclude_holidays=true,
      escalation_rules=jsonb_build_object('thresholds', jsonb_build_array()),
      is_active=true, priority_score=30,
      updated_at=NOW()
    WHERE id=v_id;
  END IF;

  -- 服务请求-标准
  SELECT id INTO v_id FROM sla_policies WHERE tenant_id=v_tenant_id AND name='服务请求-标准';
  IF v_id IS NULL THEN
    INSERT INTO sla_policies(name,description,customer_tier,ticket_type,priority,
      response_time_minutes,resolution_time_minutes,business_hours,
      exclude_weekends,exclude_holidays,escalation_rules,
      is_active,priority_score,tenant_id,created_at,updated_at)
    VALUES ('服务请求-标准',
      '标准服务请求 SLA——账户开通、权限申请、资源交付等典型请求。',
      'silver','service_request','medium',240,2880,
      jsonb_build_object('work_days', jsonb_build_array(1,2,3,4,5),
                         'start_time','09:00','end_time','18:00','time_zone','Asia/Shanghai'),
      true,true,
      jsonb_build_object('thresholds', jsonb_build_array(
        jsonb_build_object('percent', 80, 'notify', jsonb_build_array('sd_manager'))
      )),
      true, 40, v_tenant_id, NOW(), NOW());
  END IF;

  -- 变更-标准
  SELECT id INTO v_id FROM sla_policies WHERE tenant_id=v_tenant_id AND name='变更-标准';
  IF v_id IS NULL THEN
    INSERT INTO sla_policies(name,description,customer_tier,ticket_type,priority,
      response_time_minutes,resolution_time_minutes,business_hours,
      exclude_weekends,exclude_holidays,escalation_rules,
      is_active,priority_score,tenant_id,created_at,updated_at)
    VALUES ('变更-标准',
      '标准/紧急变更 SLA——评估、审批、实施、验证全周期跟踪。',
      'gold','change','high',60,1440,
      jsonb_build_object('work_days', jsonb_build_array(1,2,3,4,5,6,7),
                         'start_time','08:00','end_time','22:00','time_zone','Asia/Shanghai'),
      false,false,
      jsonb_build_object('thresholds', jsonb_build_array(
        jsonb_build_object('percent', 80, 'notify', jsonb_build_array('ops_manager','it_director'))
      )),
      true, 70, v_tenant_id, NOW(), NOW());
  END IF;

  -- 问题-标准
  SELECT id INTO v_id FROM sla_policies WHERE tenant_id=v_tenant_id AND name='问题-标准';
  IF v_id IS NULL THEN
    INSERT INTO sla_policies(name,description,customer_tier,ticket_type,priority,
      response_time_minutes,resolution_time_minutes,business_hours,
      exclude_weekends,exclude_holidays,escalation_rules,
      is_active,priority_score,tenant_id,created_at,updated_at)
    VALUES ('问题-标准',
      '问题管理 SLA——根因分析、绕过方案与已知错误沉淀。',
      'silver','problem','medium',240,4320,
      jsonb_build_object('work_days', jsonb_build_array(1,2,3,4,5),
                         'start_time','09:00','end_time','18:00','time_zone','Asia/Shanghai'),
      true,true,
      jsonb_build_object('thresholds', jsonb_build_array()),
      true, 50, v_tenant_id, NOW(), NOW());
  END IF;
END $$;

-- ---------------------------------------------------------------------
-- 2) 知识库 SOP 文章（12 篇）
-- ---------------------------------------------------------------------
-- 全部通过标题唯一锚定，重复执行只更新 content/category/tags/is_published，
-- 不创建重复条目。

DO $$
DECLARE
  v_tenant_id bigint := 1;
  v_author_id bigint;
  v_id bigint;

  C_KB_INCIDENT constant text :=
    '# 事件响应标准流程 (Incident Response SOP)

## 目标
- 在 SLA 时间窗内恢复业务可用性，最小化业务影响。
- 事件处理全过程留痕，作为后续问题/变更的输入。

## 适用范围
- 服务台一线、二线、三线；
- IT 运维、应用支持、网络、DBA、信息安全等业务团队。

## 流程步骤

### 1. 受理与分类（5 分钟内）
- 服务台接到工单后立即在 ITSM 创建事件单（Incident），分配工单号；
- 询问用户：什么业务受影响、影响范围、错误信息、最近变更；
- 初步评估影响面和紧急度：
  - 影响面 = 受影响用户数 / 关键业务
  - 紧急度 = 业务不可用程度
  - 优先级 = max(紧急度, 客户等级映射紧急度)

### 2. 分派与升级
- 根据 SLA 策略自动绑定响应责任人；
- 触发即时通知（IM/邮件/短信）给一线工程师；
- 当 SLA 阈值达到 80% 时自动触发升级：升级至二线、运维经理、IT 总监。

### 3. 处置与沟通
- 处置过程必须定时更新工单状态（in_progress）；
- 每 30 分钟在事件中追加进展，哪怕"无进展的进展"；
- 出现 P0/P1 须在 10 分钟内通知业务方。

### 4. 关闭与回顾
- 关闭前必须：
  - 与用户确认业务恢复；
  - 记录恢复时间、影响时长；
  - 关联相关变更、CMDB CI、知识文章；
- P0 重大事件须在 5 个工作日内提交 PIR（Post-Incident Review）。

## 注意事项
- 严禁静默关闭——所有事件必须可追踪；
- SLA 时间以权威时间戳为准，不得依赖 UI 显示；
- 涉及多团队的工单必须指定 single point of contact（SPOC）。';
BEGIN
  SELECT author_id INTO v_author_id FROM _seed_ctx;
  -- 1. event response
  SELECT id INTO v_id FROM knowledge_articles WHERE tenant_id=v_tenant_id AND title='事件响应标准流程 (Incident Response SOP)';
  IF v_id IS NULL THEN
    INSERT INTO knowledge_articles(title,content,category,tags,author_id,tenant_id,is_published,view_count,like_count,created_at,updated_at)
    VALUES ('事件响应标准流程 (Incident Response SOP)', C_KB_INCIDENT, 'Incident',
            'incident,sop,response,oncall', v_author_id, v_tenant_id, true, 0, 0, NOW(), NOW());
  ELSE
    UPDATE knowledge_articles SET content=C_KB_INCIDENT, category='Incident',
      tags='incident,sop,response,oncall', is_published=true, author_id=v_author_id,
      updated_at=NOW()
    WHERE id=v_id;
  END IF;
END $$;

DO $$
DECLARE
  v_tenant_id bigint := 1;
  v_author_id bigint;
  v_id bigint;
  C_CONTENT constant text :=
    '# 密码重置与账号解锁 SOP

## 目标
统一密码重置 / 账号解锁 / MFA 重新绑定流程，确保身份可恢复且审计可追溯。

## 触发场景
- 用户忘记密码；
- 账号因登录失败次数过多被锁定；
- MFA 设备丢失或重新绑定；
- 安全事件触发的强制重置。

## 受理步骤

### 1. 身份核验（必做）
- 必须通过以下至少 2 项核验：
  - 工号 / 邮箱 / 手机号；
  - 最近登录设备或地点；
  - 直接经理或指定代理人核实；
- 高敏感角色（财务、HR、基础设施）必须由其主管书面授权。

### 2. 创建服务请求
- 模板选择：IT / 账户开通 / 密码重置；
- 必填字段：
  - 申请人、邮箱、所属部门；
  - 核验方式与时间；
  - 授权人姓名与审批流水号。

### 3. 处置
- 服务台统一通过 IdP 控制台重置；
- 一次性临时密码通过加密信道送达，禁止明文邮件；
- 立即要求首次登录强制修改；
- 保留审计日志：操作人、操作时间、源 IP。

### 4. 关闭
- 用户确认账号可用；
- 工单关闭前必须删除一次性凭证；
- 高敏感账号变更须追加到 IAM 审计平台。

## 注意事项
- 严禁以聊天工具直接发送最终密码；
- 每一次重置都必须留痕；
- 同一账号 24 小时内第三次重置自动升级至安全团队。';
BEGIN
  SELECT author_id INTO v_author_id FROM _seed_ctx;
  SELECT id INTO v_id FROM knowledge_articles WHERE tenant_id=v_tenant_id AND title='密码重置与账号解锁 SOP';
  IF v_id IS NULL THEN
    INSERT INTO knowledge_articles(title,content,category,tags,author_id,tenant_id,is_published,view_count,like_count,created_at,updated_at)
    VALUES ('密码重置与账号解锁 SOP', C_CONTENT, 'ServiceRequest',
            'service_request,account,sop', v_author_id, v_tenant_id, true, 0, 0, NOW(), NOW());
  ELSE
    UPDATE knowledge_articles SET content=C_CONTENT, category='ServiceRequest',
      tags='service_request,account,sop', is_published=true, author_id=v_author_id,
      updated_at=NOW()
    WHERE id=v_id;
  END IF;
END $$;

DO $$
DECLARE
  v_tenant_id bigint := 1;
  v_author_id bigint;
  v_id bigint;
  C_CONTENT constant text :=
    '# 服务器故障应急处理 SOP

## 目标
标准化服务器软硬件故障应急处置，缩短恢复时间。

## 适用对象
- 物理服务器、虚拟机、容器节点；
- 所有生产、准生产环境的 Linux/Windows 主机。

## 1. 判定故障范围
- 查看监控：CPU / Memory / Disk / Network / Service Port；
- 查看告警：Zabbix / Prometheus / 云监控；
- 与受影响应用 owner 联系，确认是否影响业务。

## 2. 紧急止血
- 应用层：如为单实例，先通过 LB 摘除故障节点；
- 系统层：
  - OOM / Disk Full：清理日志、临时文件；
  - Kernel Panic：保留 coredump 后强制重启；
  - 硬件故障：切换至备用节点后下电检查。
- 数据层：禁止在故障态下进行 DDL/DML，先切只读再决策。

## 3. 升级与通报
- P0/P1 自动触发管理层通知；
- 涉及数据丢失、可能违规时立即上报法务/合规接口人。

## 4. 恢复验证
- 服务端口 / 健康检查 / 业务冒烟；
- SLA 计时以"业务恢复可服务"为终点；
- 完成后立即在事件工单中追加"恢复时间戳"。

## 5. 复盘
- 涉及硬件更换、Kernel 升级、配置变更的事件必须产出 PIR；
- 输出候选 RCA，必要时转入 Problem。';
BEGIN
  SELECT author_id INTO v_author_id FROM _seed_ctx;
  SELECT id INTO v_id FROM knowledge_articles WHERE tenant_id=v_tenant_id AND title='服务器故障应急处理 SOP';
  IF v_id IS NULL THEN
    INSERT INTO knowledge_articles(title,content,category,tags,author_id,tenant_id,is_published,view_count,like_count,created_at,updated_at)
    VALUES ('服务器故障应急处理 SOP', C_CONTENT, 'Incident',
            'incident,server,emergency,sop', v_author_id, v_tenant_id, true, 0, 0, NOW(), NOW());
  ELSE
    UPDATE knowledge_articles SET content=C_CONTENT, category='Incident',
      tags='incident,server,emergency,sop', is_published=true, author_id=v_author_id,
      updated_at=NOW()
    WHERE id=v_id;
  END IF;
END $$;

DO $$
DECLARE
  v_tenant_id bigint := 1;
  v_author_id bigint;
  v_id bigint;
  C_CONTENT constant text :=
    '# 数据库备份与恢复 SOP

## 目标
保证关键数据库可恢复、可验证，并满足监管合规要求。

## 适用范围
- PostgreSQL、MySQL、ClickHouse、MongoDB 等公司核心库；
- RPO ≤ 1 小时、RTO ≤ 4 小时的高可用等级。

## 1. 备份策略
- 全量备份：每周日 02:00；
- 增量备份：每日 02:00（除全量日）；
- binlog / WAL：实时归档；
- 校验：每月最后一个周六做一次恢复演练。

## 2. 备份前检查
- 备份工具版本、磁盘空间、目标存储可用性；
- 数据库当前连接数、长事务、复制延迟；
- 关联应用的变更窗口与沟通通告。

## 3. 执行备份（变更单执行）
- 必须走变更审批流程（CHG-…）；
- 备份完成后立即做 checksum 验证；
- 备份元数据写入备份调度平台。

## 4. 恢复演练（季度必做）
- 在隔离环境中拉起最近一次全量 + binlog 增量；
- 验证关键表行数 / 抽样数据一致性；
- 输出恢复时间报告，更新 RTO。

## 5. 应急恢复
- 仅在数据丢失 / 损坏场景下执行；
- 必须由 DBA + SRE 双签；
- 恢复前自动快照当前实例。

## 注意事项
- 备份密钥必须按密钥管理 SOP 存储；
- 不要在容器 / 临时卷上保存唯一副本；
- 涉及跨境数据时遵循数据驻留 SOP。';
BEGIN
  SELECT author_id INTO v_author_id FROM _seed_ctx;
  SELECT id INTO v_id FROM knowledge_articles WHERE tenant_id=v_tenant_id AND title='数据库备份与恢复 SOP';
  IF v_id IS NULL THEN
    INSERT INTO knowledge_articles(title,content,category,tags,author_id,tenant_id,is_published,view_count,like_count,created_at,updated_at)
    VALUES ('数据库备份与恢复 SOP', C_CONTENT, 'Change',
            'change,database,backup,sop', v_author_id, v_tenant_id, true, 0, 0, NOW(), NOW());
  ELSE
    UPDATE knowledge_articles SET content=C_CONTENT, category='Change',
      tags='change,database,backup,sop', is_published=true, author_id=v_author_id,
      updated_at=NOW()
    WHERE id=v_id;
  END IF;
END $$;

DO $$
DECLARE
  v_tenant_id bigint := 1;
  v_author_id bigint;
  v_id bigint;
  C_CONTENT constant text :=
    '# 网络故障排查流程

## 目标
减少网络故障定位时间，明确分段责任与升级路径。

## 适用范围
- 办公网、数据中心局域网、广域网、跨云专线；
- DNS、DHCP、负载均衡、VPN、防火墙。

## 1. 初步定位（10 分钟内）
- 用户报障 → 看监控：链路、出口、设备 CPU / 内存；
- 多人 / 多区域同时报障 → 平台级故障，直接进入全局事件；
- 单点 / 单区域 → 进入 L1 排查。

## 2. L1 分段定位
| 段落 | 检查项 |
|------|--------|
| 终端 / LAN | 网线、DHCP、ACL、802.1X |
| 内部交换 | 端口状态、STP、VLAN、ARP |
| 内部路由 | OSPF/BGP 会话、路由表、ACL |
| 出口 | NAT、QoS、ISP、BGP |
| DNS / DHCP | 解析、租约、源 IP 池 |

## 3. 工具辅助
- mtr / iperf / tcpdump；
- 监控：SNMP / sFlow / NetFlow；
- 配置：Ansible / Config Diff。

## 4. 升级与协作
- 涉及 ISP、云厂商、跨域链路：在 30 分钟内拉群；
- 涉及核心交换机 / 防火墙：必须在变更窗口操作，否则按紧急变更处理。

## 5. 关闭与复盘
- 与用户确认恢复；
- 文档化路径 / 影响面，纳入 CMDB 拓扑；
- 复盘聚焦：分段检测、监控盲区、自动化机会。';
BEGIN
  SELECT author_id INTO v_author_id FROM _seed_ctx;
  SELECT id INTO v_id FROM knowledge_articles WHERE tenant_id=v_tenant_id AND title='网络故障排查流程 (Network Troubleshooting SOP)';
  IF v_id IS NULL THEN
    INSERT INTO knowledge_articles(title,content,category,tags,author_id,tenant_id,is_published,view_count,like_count,created_at,updated_at)
    VALUES ('网络故障排查流程 (Network Troubleshooting SOP)', C_CONTENT, 'Incident',
            'incident,network,troubleshooting,sop', v_author_id, v_tenant_id, true, 0, 0, NOW(), NOW());
  ELSE
    UPDATE knowledge_articles SET content=C_CONTENT, category='Incident',
      tags='incident,network,troubleshooting,sop', is_published=true, author_id=v_author_id,
      updated_at=NOW()
    WHERE id=v_id;
  END IF;
END $$;

DO $$
DECLARE
  v_tenant_id bigint := 1;
  v_author_id bigint;
  v_id bigint;
  C_CONTENT constant text :=
    '# 服务请求履行规范

## 目标
保证服务请求履行效率、可预期、可审计。

## 适用范围
- IT 服务目录内的全部条目；
- 涵盖账户开通、资源申请、权限变更、办公支持等。

## 1. 工单分类
- 类型：service_request；
- 子类必须从目录（service_catalog）选择，禁止自由文本。

## 2. 审批
- 高价值 / 高风险请求必须经过 BPMN 审批链；
- 审批超时按 SLA 阈值告警；
- 申请人 / 审批人不得为同一人。

## 3. 履行
- 通过 CMDB 校验目标资源是否存在；
- 涉及自动化交付的请求必须保留脚本版本与执行日志；
- 涉及手工操作的请求必须在工单中标注操作步骤与回滚点。

## 4. 交付 / 验证
- 交付完成后通知用户；
- 用户在工单中确认或反馈；
- 默认 7 天后无反馈自动关闭，附用户告知邮件。

## 5. 关闭
- 工单必须关联 CMDB 变更日志；
- 服务请求数据用于 SLA 与满意度报表。';
BEGIN
  SELECT author_id INTO v_author_id FROM _seed_ctx;
  SELECT id INTO v_id FROM knowledge_articles WHERE tenant_id=v_tenant_id AND title='服务请求履行规范 (Service Request Fulfillment SOP)';
  IF v_id IS NULL THEN
    INSERT INTO knowledge_articles(title,content,category,tags,author_id,tenant_id,is_published,view_count,like_count,created_at,updated_at)
    VALUES ('服务请求履行规范 (Service Request Fulfillment SOP)', C_CONTENT, 'ServiceRequest',
            'service_request,fulfillment,sop', v_author_id, v_tenant_id, true, 0, 0, NOW(), NOW());
  ELSE
    UPDATE knowledge_articles SET content=C_CONTENT, category='ServiceRequest',
      tags='service_request,fulfillment,sop', is_published=true, author_id=v_author_id,
      updated_at=NOW()
    WHERE id=v_id;
  END IF;
END $$;

DO $$
DECLARE
  v_tenant_id bigint := 1;
  v_author_id bigint;
  v_id bigint;
  C_CONTENT constant text :=
    '# SLA 升级流程

## 目标
在 SLA 时间窗内及时识别逾期风险，按预设路径升级告警。

## 阈值梯度
| 阈值 | 通知对象 | 通讯渠道 |
|------|----------|----------|
| 50%  | 一线 owner、SD manager | IM、Email |
| 80%  | 二线 owner、ops_manager | IM、Email、SMS |
| 100% | ops_director、it_director | IM、Email、SMS、电话 |
| 130% | dept_manager、CTO 办公室 | 全部渠道 |

## 升级触发
- 任一阈值跨越即升级；
- 在多个阈值同时跨越时，仅触发最高级；
- 关闭工单时清除所有待发送告警。

## 工单与告警一致性
- 告警必须引用工单编号；
- 不可对外泄漏敏感正文，仅展示 SLA 摘要；
- 告警平台投递失败须记录并重试 3 次后升级到 IM 值班群。

## 注意事项
- 节假日调整须事先在 SLA policy 中声明；
- SLA 不允许在工单中"覆盖式"修改——只能通过 policy 升级；
- 越权修改 SLA 视为违规事件，写入审计。';
BEGIN
  SELECT author_id INTO v_author_id FROM _seed_ctx;
  SELECT id INTO v_id FROM knowledge_articles WHERE tenant_id=v_tenant_id AND title='SLA 升级流程 (SLA Escalation Workflow)';
  IF v_id IS NULL THEN
    INSERT INTO knowledge_articles(title,content,category,tags,author_id,tenant_id,is_published,view_count,like_count,created_at,updated_at)
    VALUES ('SLA 升级流程 (SLA Escalation Workflow)', C_CONTENT, 'SLA',
            'sla,escalation,workflow', v_author_id, v_tenant_id, true, 0, 0, NOW(), NOW());
  ELSE
    UPDATE knowledge_articles SET content=C_CONTENT, category='SLA',
      tags='sla,escalation,workflow', is_published=true, author_id=v_author_id,
      updated_at=NOW()
    WHERE id=v_id;
  END IF;
END $$;

DO $$
DECLARE
  v_tenant_id bigint := 1;
  v_author_id bigint;
  v_id bigint;
  C_CONTENT constant text :=
    '# 问题管理与根本原因分析 SOP

## 目标
把反复发生的事件归并为问题，定位根因并避免复发。

## 触发条件
- 同类事件 30 天内出现 ≥ 3 次；
- P0/P1 事件的影响面广；
- RCA 要求（合规、客户审计、年度回顾）。

## 生命周期
new → under_review → rca_in_progress → workaround_published → known_error → resolved → closed

### 1. 立问题单
- 由事件经理、SRE、二线发起；
- 关联已发生的事件工单；
- 复制受影响 CI 列表到问题单。

### 2. RCA
- 推荐方法：
  - 5 Whys（适用于单链路故障）；
  - Fishbone（适用于多因子耦合）；
  - Kepner-Tregoe（高风险决策场景）；
- 输出：根因（机制，非现象）、触发条件、检测手段缺失。

### 3. Workaround
- 在 RCA 完成前必须提供临时绕过方案；
- 必要时发布已知错误（Known Error）到知识库；
- 当 workaround 足够稳定时可关闭问题单，但保留 known_error 提醒。

### 4. 解决与预防
- 给出永久解决方案（变更 / 架构调整 / 流程修订）；
- 在变更单实施后跟踪 30 天确认无复发。

## 注意事项
- 不要把问题当作"事件+"；
- 不要在根因未确定前关闭问题；
- 重大问题必须由 IT 总监签发。';
BEGIN
  SELECT author_id INTO v_author_id FROM _seed_ctx;
  SELECT id INTO v_id FROM knowledge_articles WHERE tenant_id=v_tenant_id AND title='问题管理与根本原因分析 SOP';
  IF v_id IS NULL THEN
    INSERT INTO knowledge_articles(title,content,category,tags,author_id,tenant_id,is_published,view_count,like_count,created_at,updated_at)
    VALUES ('问题管理与根本原因分析 SOP', C_CONTENT, 'Problem',
            'problem,rca,known_error,sop', v_author_id, v_tenant_id, true, 0, 0, NOW(), NOW());
  ELSE
    UPDATE knowledge_articles SET content=C_CONTENT, category='Problem',
      tags='problem,rca,known_error,sop', is_published=true, author_id=v_author_id,
      updated_at=NOW()
    WHERE id=v_id;
  END IF;
END $$;

DO $$
DECLARE
  v_tenant_id bigint := 1;
  v_author_id bigint;
  v_id bigint;
  C_CONTENT constant text :=
    '# CMDB 变更管理规范

## 目标
保证 CMDB 数据权威、可追溯、能反映真实系统。

## 数据分类
| 类别 | 责任方 | 更新触发 |
|------|--------|----------|
| CI 类型 / 属性 schema | CMDB 平台团队 | 主版本变更 |
| CI 实例 | 各业务系统 owner | 上线 / 下线 / 配置变更 |
| CI 关系 | 业务系统 owner + SRE | 部署拓扑变更 |
| 关系类型 | CMDB 平台团队 | 元模型变更 |

## 变更流程

### 1. 计划阶段
- CMDB 变更纳入变更审批；
- 大批量变更（>100）必须走离线导入任务，并通过 dry-run 校验。

### 2. 执行阶段
- 必须使用 API / ETL，不允许直接 SQL 写入；
- 关系新增必须双端校验（source、target CI 都存在）；
- 复杂关系必须给出影响面报告。

### 3. 验证
- 完整性：CMDB 与实际系统拓扑抽样一致；
- 唯一性：name + tenant 唯一；
- 关系闭环：无悬挂 source/target 节点。

### 4. 回滚
- 变更失败时仅可删除由本次变更引入的记录；
- 禁止物理删除被运行实例引用的 CI。

## 注意事项
- CI 必须有 owner，无 owner 的 CI 进入待整改队列；
- CI 必须有生命周期（online / standby / retired）；
- 跨租户的 CI 必须显式标注，禁止自动全网拉群。';
BEGIN
  SELECT author_id INTO v_author_id FROM _seed_ctx;
  SELECT id INTO v_id FROM knowledge_articles WHERE tenant_id=v_tenant_id AND title='CMDB 变更管理规范 (CMDB Change Governance)';
  IF v_id IS NULL THEN
    INSERT INTO knowledge_articles(title,content,category,tags,author_id,tenant_id,is_published,view_count,like_count,created_at,updated_at)
    VALUES ('CMDB 变更管理规范 (CMDB Change Governance)', C_CONTENT, 'CMDB',
            'cmdb,governance,change,sop', v_author_id, v_tenant_id, true, 0, 0, NOW(), NOW());
  ELSE
    UPDATE knowledge_articles SET content=C_CONTENT, category='CMDB',
      tags='cmdb,governance,change,sop', is_published=true, author_id=v_author_id,
      updated_at=NOW()
    WHERE id=v_id;
  END IF;
END $$;

DO $$
DECLARE
  v_tenant_id bigint := 1;
  v_author_id bigint;
  v_id bigint;
  C_CONTENT constant text :=
    '# 知识库文章编写与发布规范

## 目标
输出结构清晰、可检索、可信赖的 SOP / Runbook / FAQ。

## 适用对象
- 服务台、技术支持、应用 owner、SRE、安全团队；
- 任何向外/向内供其他团队检索的运营知识。

## 文章结构模板
1. 目标与适用场景
2. 前置条件 / 角色权限
3. 步骤（编号 + 子步骤）
4. 异常处理与升级路径
5. 注意事项与边界
6. 参考资料 / 关联 CMDB / 关联工单

## 写作要求
- 标题具象、可被搜索：包含动词 + 对象 + 场景；
- 第一段写"做什么、什么时候用"；
- 步骤段落使用祈使语气；
- 列表使用有序列表，便于复读；
- 截图必须配红线/序号；
- 关联工单 / 变更 / CI 用稳定编号引用。

## 评审与发布
- 一级作者（自己认领）→ 同团队复核 → 跨团队技术评审 → 发布；
- P0 / 重大事件类文章必须由对应系统 owner 复核；
- 每季度抽样复核，标记过期 / 仍有效。

## 注意事项
- 严禁出现真实账号、密码、API Key、内部域名；
- 合规敏感字段（金融、医疗、个人信息）禁止在正文中留例；
- 中文 SOP 必须给出英文标题/关键词，提升检索。';
BEGIN
  SELECT author_id INTO v_author_id FROM _seed_ctx;
  SELECT id INTO v_id FROM knowledge_articles WHERE tenant_id=v_tenant_id AND title='知识库文章编写与发布规范';
  IF v_id IS NULL THEN
    INSERT INTO knowledge_articles(title,content,category,tags,author_id,tenant_id,is_published,view_count,like_count,created_at,updated_at)
    VALUES ('知识库文章编写与发布规范', C_CONTENT, 'KnowledgeBase',
            'knowledge_base,authoring,guideline', v_author_id, v_tenant_id, true, 0, 0, NOW(), NOW());
  ELSE
    UPDATE knowledge_articles SET content=C_CONTENT, category='KnowledgeBase',
      tags='knowledge_base,authoring,guideline', is_published=true, author_id=v_author_id,
      updated_at=NOW()
    WHERE id=v_id;
  END IF;
END $$;

DO $$
DECLARE
  v_tenant_id bigint := 1;
  v_author_id bigint;
  v_id bigint;
  C_CONTENT constant text :=
    '# SLA 客户等级与优先级对照表

## 客户等级
| 等级 | 含义 | 示例 |
|------|------|------|
| platinum | 战略级，7x24 即时响应 | 关键业务 / 高商业影响 |
| gold | 重要级，工作时段优先 | 大型业务 / 重要业务 |
| silver | 标准级，工作时段标准 | 中小业务 |
| bronze | 经济级，工作时段经济 | 一般内部业务 |

## 优先级映射
| 优先级 | 中文 | 响应时间 | 解决时间 |
|--------|------|----------|----------|
| P0 | 致命 | 5 分钟 | 60 分钟 |
| P1 | 紧急 | 15 分钟 | 4 小时 |
| P2 | 标准 | 30 分钟 | 8 小时 |
| P3 | 一般 | 2 小时 | 24 小时 |
| P4 | 信息 | 8 小时 | 5 个工作日 |

## 工单类型映射
- incident / problem：以上 SLA；
- service_request：按服务目录中明示的履行时长；
- change：按变更等级（标准 / 紧急 / 重大）映射；
- 工单优先级由"客户等级 × 紧急度 × 影响面"共同决定。';
BEGIN
  SELECT author_id INTO v_author_id FROM _seed_ctx;
  SELECT id INTO v_id FROM knowledge_articles WHERE tenant_id=v_tenant_id AND title='SLA 客户等级与优先级对照表';
  IF v_id IS NULL THEN
    INSERT INTO knowledge_articles(title,content,category,tags,author_id,tenant_id,is_published,view_count,like_count,created_at,updated_at)
    VALUES ('SLA 客户等级与优先级对照表', C_CONTENT, 'SLA',
            'sla,customer_tier,reference', v_author_id, v_tenant_id, true, 0, 0, NOW(), NOW());
  ELSE
    UPDATE knowledge_articles SET content=C_CONTENT, category='SLA',
      tags='sla,customer_tier,reference', is_published=true, author_id=v_author_id,
      updated_at=NOW()
    WHERE id=v_id;
  END IF;
END $$;

DO $$
DECLARE
  v_tenant_id bigint := 1;
  v_author_id bigint;
  v_id bigint;
  C_CONTENT constant text :=
    '# 重大事件通报模板 (Major Incident Communication Template)

## 通知对象
- 受影响业务方代表；
- 内部 SLA 升级链（ops_manager → it_director → CTO 办公室）；
- 公关 / 法务（涉及对外影响时）；
- 客户成功 / 销售（涉及大客户时）。

## 通报节奏
- 首次通报：T+5 分钟内；
- 进展通报：T+15 / 30 / 60 分钟，直至恢复；
- 恢复通报：T+10 分钟内；
- 复盘通报：5 个工作日内。

## 通报模板字段
- 事件 ID：MI-<yyyyMMdd>-<seq>
- 开始时间 / 当前状态
- 影响面：受影响用户 / 业务 / 地域
- 当前进展：已确认 / 已止血 / 未恢复
- 下次通报时间
- 联系人：SPOC 姓名、IM、电话

## 注意事项
- 通报内容不得包含客户隐私、凭据、源代码；
- 同一事件只能由 SPOC 对外发声；
- 通报模板必须自动追加到事件工单关联。';
BEGIN
  SELECT author_id INTO v_author_id FROM _seed_ctx;
  SELECT id INTO v_id FROM knowledge_articles WHERE tenant_id=v_tenant_id AND title='重大事件通报模板 (Major Incident Communication Template)';
  IF v_id IS NULL THEN
    INSERT INTO knowledge_articles(title,content,category,tags,author_id,tenant_id,is_published,view_count,like_count,created_at,updated_at)
    VALUES ('重大事件通报模板 (Major Incident Communication Template)', C_CONTENT, 'Incident',
            'incident,major_incident,communication,template', v_author_id, v_tenant_id, true, 0, 0, NOW(), NOW());
  ELSE
    UPDATE knowledge_articles SET content=C_CONTENT, category='Incident',
      tags='incident,major_incident,communication,template', is_published=true, author_id=v_author_id,
      updated_at=NOW()
    WHERE id=v_id;
  END IF;
END $$;

-- ---------------------------------------------------------------------
-- 3) CMDB CI 实例（16 条）
-- ---------------------------------------------------------------------
-- 稳定键 (tenant_id, serial_number) 有唯一索引，可直接 UPSERT。
-- ci_type 通过 name 反查 ci_types.id。

INSERT INTO configuration_items
  (name,description,ci_type_id,ci_type,status,environment,criticality,
   asset_tag,serial_number,model,vendor,location,
   assigned_to,owned_by,ownership_mode,source,attributes,
   tenant_id,version,created_at,updated_at,lifecycle_status)
VALUES
  -- server (4)
  ('web-prod-01','Web 接入层主机，承担全球用户登录与静态资源分发',
   (SELECT id FROM ci_types WHERE tenant_id=1 AND name='server' LIMIT 1),
   'server','active','production','high',
   'AST-100001','SN-WEB-0001','PowerEdge R750','Dell','DC-Shanghai-A / Rack R03',
   'l2_support','ops_manager','managed','manual','{}'::jsonb,
   1,1,NOW(),NOW(),'online'),
  ('web-prod-02','Web 接入层主机，与 web-prod-01 互备',
   (SELECT id FROM ci_types WHERE tenant_id=1 AND name='server' LIMIT 1),
   'server','active','production','high',
   'AST-100002','SN-WEB-0002','PowerEdge R750','Dell','DC-Shanghai-A / Rack R03',
   'l2_support','ops_manager','managed','manual','{}'::jsonb,
   1,1,NOW(),NOW(),'online'),
  ('app-prod-01','业务应用主机，部署订单/支付核心服务',
   (SELECT id FROM ci_types WHERE tenant_id=1 AND name='server' LIMIT 1),
   'server','active','production','critical',
   'AST-100010','SN-APP-0001','PowerEdge R650','Dell','DC-Shanghai-A / Rack R05',
   'l3_expert','ops_director','managed','manual','{}'::jsonb,
   1,1,NOW(),NOW(),'online'),
  ('app-prod-02','业务应用主机，承担客服/工作流',
   (SELECT id FROM ci_types WHERE tenant_id=1 AND name='server' LIMIT 1),
   'server','active','production','high',
   'AST-100011','SN-APP-0002','PowerEdge R650','Dell','DC-Shanghai-A / Rack R05',
   'l3_expert','ops_director','managed','manual','{}'::jsonb,
   1,1,NOW(),NOW(),'online'),
  -- database (2)
  ('pg-main','主 PostgreSQL 集群，含核心订单/支付数据',
   (SELECT id FROM ci_types WHERE tenant_id=1 AND name='database' LIMIT 1),
   'database','active','production','critical',
   'AST-200001','SN-PG-0001','PostgreSQL 16','PostgreSQL','DC-Shanghai-B / Rack R10',
   'dba','dba','managed','manual',
   '{"version":"16.4","replication":"streaming","shards":4,"snapshot_hr":1}'::jsonb,
   1,1,NOW(),NOW(),'online'),
  ('redis-cache','Redis 集群，承担登录态、限流、缓存',
   (SELECT id FROM ci_types WHERE tenant_id=1 AND name='database' LIMIT 1),
   'database','active','production','high',
   'AST-200002','SN-RDS-0001','Redis 7 Cluster','Redis','DC-Shanghai-B / Rack R11',
   'l2_support','dba','managed','manual','{}'::jsonb,
   1,1,NOW(),NOW(),'online'),
  -- network (2)
  ('fw-edge-01','互联网边界防火墙，主备双机',
   (SELECT id FROM ci_types WHERE tenant_id=1 AND name='network' LIMIT 1),
   'network','active','production','critical',
   'AST-300001','SN-FW-0001','PA-3220','Palo Alto','DC-Shanghai-A / Rack R01',
   'network_eng','network_eng','managed','manual','{}'::jsonb,
   1,1,NOW(),NOW(),'online'),
  ('lb-nginx-01','七层负载均衡，业务流量入口',
   (SELECT id FROM ci_types WHERE tenant_id=1 AND name='network' LIMIT 1),
   'network','active','production','critical',
   'AST-300010','SN-LB-0001','F5 BIG-IP 5200v','F5','DC-Shanghai-A / Rack R02',
   'l2_support','network_eng','managed','manual','{}'::jsonb,
   1,1,NOW(),NOW(),'online'),
  -- storage (2)
  ('s3-media','对象存储，存放静态资源与备份归档',
   (SELECT id FROM ci_types WHERE tenant_id=1 AND name='storage' LIMIT 1),
   'storage','active','production','medium',
   'AST-400001','SN-OSS-0001','S3 Compatible','MinIO','DC-Shanghai-A',
   'l2_support','ops_manager','managed','manual','{}'::jsonb,
   1,1,NOW(),NOW(),'online'),
  ('nfs-shared','内部共享存储，承载构建产物与日志归档',
   (SELECT id FROM ci_types WHERE tenant_id=1 AND name='storage' LIMIT 1),
   'storage','active','production','low',
   'AST-400010','SN-NFS-0001','NetApp FAS2750','NetApp','DC-Shanghai-B / Rack R12',
   'l2_support','ops_manager','managed','manual','{}'::jsonb,
   1,1,NOW(),NOW(),'online'),
  -- application (2)
  ('order-service','订单核心服务，关键链路',
   (SELECT id FROM ci_types WHERE tenant_id=1 AND name='application' LIMIT 1),
   'application','active','production','critical',
   'AST-500001','SN-APP-SVC-001','JVM/Spring Boot 3','Internal','k8s-prod-cluster / ns order',
   'developer','rd_manager','managed','discovery',
   '{"deploy_strategy":"rolling","replicas":6,"language":"java"}'::jsonb,
   1,1,NOW(),NOW(),'online'),
  ('payment-service','支付/清算服务，与银行网关通信',
   (SELECT id FROM ci_types WHERE tenant_id=1 AND name='application' LIMIT 1),
   'application','active','production','critical',
   'AST-500010','SN-APP-SVC-002','Go 1.22','Internal','k8s-prod-cluster / ns payment',
   'developer','rd_manager','managed','discovery',
   '{"deploy_strategy":"blue-green","replicas":4,"language":"go"}'::jsonb,
   1,1,NOW(),NOW(),'online'),
  -- middleware (2)
  ('kafka-cluster','Kafka 集群，承担事件流、CDC、审计',
   (SELECT id FROM ci_types WHERE tenant_id=1 AND name='middleware' LIMIT 1),
   'middleware','active','production','high',
   'AST-600001','SN-KFK-0001','Kafka 3.6','Confluent','DC-Shanghai-B / Rack R13',
   'l3_expert','ops_director','managed','manual','{}'::jsonb,
   1,1,NOW(),NOW(),'online'),
  ('rabbitmq-broker','内部任务/通知消息总线',
   (SELECT id FROM ci_types WHERE tenant_id=1 AND name='middleware' LIMIT 1),
   'middleware','active','production','medium',
   'AST-600010','SN-RMQ-0001','RabbitMQ 3.13','RabbitMQ','DC-Shanghai-B / Rack R14',
   'l2_support','ops_manager','managed','manual','{}'::jsonb,
   1,1,NOW(),NOW(),'online'),
  -- cloud_vm (1)
  ('ecs-business-01','阿里云 ECS，承载 CRM / 报表',
   (SELECT id FROM ci_types WHERE tenant_id=1 AND name='cloud_vm' LIMIT 1),
   'cloud_vm','active','production','medium',
   'AST-700001','SN-ECS-cn-sh-0001','ecs.g6.large','Alibaba Cloud','cn-shanghai / zone-b',
   'l2_support','ops_manager','managed','discovery',
   '{"cloud_provider":"alibaba","public_ip":"10.0.0.42","vpc":"vpc-itsm-prod"}'::jsonb,
   1,1,NOW(),NOW(),'online'),
  -- kubernetes (1)
  ('k8s-prod-cluster','生产 Kubernetes 集群，业务应用主运行环境',
   (SELECT id FROM ci_types WHERE tenant_id=1 AND name='kubernetes' LIMIT 1),
   'kubernetes','active','production','critical',
   'AST-800001','SN-K8S-0001','Kubernetes 1.30','Internal','DC-Shanghai-B / multi-rack',
   'l3_expert','ops_director','managed','discovery',
   '{"nodes":9,"network_plugin":"calico","ingress":"nginx-ingress"}'::jsonb,
   1,1,NOW(),NOW(),'online')
ON CONFLICT (tenant_id, serial_number) DO UPDATE
  SET name=EXCLUDED.name,
      description=EXCLUDED.description,
      ci_type_id=EXCLUDED.ci_type_id,
      ci_type=EXCLUDED.ci_type,
      status=EXCLUDED.status,
      environment=EXCLUDED.environment,
      criticality=EXCLUDED.criticality,
      asset_tag=EXCLUDED.asset_tag,
      model=EXCLUDED.model,
      vendor=EXCLUDED.vendor,
      location=EXCLUDED.location,
      assigned_to=EXCLUDED.assigned_to,
      owned_by=EXCLUDED.owned_by,
      ownership_mode=EXCLUDED.ownership_mode,
      source=EXCLUDED.source,
      attributes=EXCLUDED.attributes,
      version=configuration_items.version+1,
      updated_at=NOW();

-- ---------------------------------------------------------------------
-- 4) CMDB CI 关系（16 条）
-- ---------------------------------------------------------------------
-- 唯一索引 (source_ci_id, target_ci_id, relationship_type) → 直接 ON CONFLICT。
-- 通过 (tenant_id, name) 子查询定位 source / target CI 的 id，避免硬编码。

INSERT INTO ci_relationships
  (tenant_id,relationship_type,strength,impact_level,is_active,is_discovered,
   description,metadata,created_at,updated_at,source_ci_id,target_ci_id)
SELECT
  1, rel.relationship_type, rel.strength, rel.impact_level, rel.is_active, rel.is_discovered,
  rel.description, '{}'::jsonb, NOW(), NOW(),
  src.id AS source_ci_id, dst.id AS target_ci_id
FROM (VALUES
  ('lb-nginx-01','web-prod-01',  'connects_to','high','high',true,false,'负载均衡将流量分发至 web-prod-01'),
  ('lb-nginx-01','web-prod-02',  'connects_to','high','high',true,false,'负载均衡将流量分发至 web-prod-02'),
  ('web-prod-01','app-prod-01',  'connects_to','high','high',true,false,'Web 接入层到业务主机的内部通信'),
  ('web-prod-02','app-prod-02',  'connects_to','medium','medium',true,false,'Web 备用节点到业务主机'),
  ('app-prod-01','pg-main',      'uses',       'critical','critical',true,false,'业务主机读写主库'),
  ('app-prod-01','redis-cache',   'uses',       'high','high',true,false,'业务主机使用 Redis 缓存与限流'),
  ('order-service','kafka-cluster','uses',      'high','high',true,false,'订单事件写入 Kafka'),
  ('payment-service','kafka-cluster','uses',    'high','high',true,false,'支付事件写入 Kafka，供对账与风控消费'),
  ('app-prod-02','rabbitmq-broker','uses',      'medium','medium',true,false,'客服/工作流使用 RabbitMQ 派发任务'),
  ('order-service','k8s-prod-cluster','runs_on','critical','critical',true,false,'订单服务运行在生产 K8s 集群'),
  ('payment-service','k8s-prod-cluster','runs_on','critical','critical',true,false,'支付服务运行在生产 K8s 集群'),
  ('ecs-business-01','kafka-cluster','connects_to','low','low',true,false,'CRM/报表 ECS 消费 Kafka 事件流'),
  ('fw-edge-01','lb-nginx-01',    'connects_to','high','high',true,false,'边界防火墙到核心 LB'),
  ('app-prod-01','nfs-shared',    'uses',      'low','low',true,false,'业务主机挂载共享存储用于构建产物'),
  ('kafka-cluster','app-prod-01', 'hosted_on', 'medium','high',true,false,'Kafka 集群宿主于业务应用机柜的物理机')
) AS rel(src_name, dst_name, relationship_type, strength, impact_level, is_active, is_discovered, description)
JOIN configuration_items src
  ON src.tenant_id=1 AND src.name=rel.src_name
JOIN configuration_items dst
  ON dst.tenant_id=1 AND dst.name=rel.dst_name
ON CONFLICT (source_ci_id, target_ci_id, relationship_type) DO UPDATE
  SET strength=EXCLUDED.strength,
      impact_level=EXCLUDED.impact_level,
      description=EXCLUDED.description,
      is_active=EXCLUDED.is_active,
      is_discovered=EXCLUDED.is_discovered,
      metadata='{}'::jsonb,
      updated_at=NOW();

-- ---------------------------------------------------------------------
-- 5) 报告：执行后状态
-- ---------------------------------------------------------------------
\echo
\echo '=========== POST-SEED SUMMARY ==========='
SELECT 'sla_policies'        AS component, count(*) AS total, count(*) FILTER (WHERE is_active) AS active
  FROM sla_policies WHERE tenant_id=1
UNION ALL
SELECT 'knowledge_articles',  count(*), count(*) FILTER (WHERE is_published)
  FROM knowledge_articles WHERE tenant_id=1
UNION ALL
SELECT 'configuration_items', count(*), count(*) FILTER (WHERE status='active')
  FROM configuration_items WHERE tenant_id=1
UNION ALL
SELECT 'ci_relationships',   count(*), count(*) FILTER (WHERE is_active)
  FROM ci_relationships WHERE tenant_id=1
ORDER BY component;

\echo
\echo '=========== KNOWLEDGE ARTICLES ==========='
SELECT id, title, category, is_published, author_id
  FROM knowledge_articles WHERE tenant_id=1
  ORDER BY id;

\echo
\echo '=========== SLA POLICIES ==========='
SELECT id, name, priority, response_time_minutes AS resp_min,
       resolution_time_minutes AS reso_min, customer_tier, ticket_type, is_active, priority_score
  FROM sla_policies WHERE tenant_id=1 ORDER BY priority_score DESC, id;

\echo
\echo '=========== CONFIGURATION ITEMS ==========='
SELECT id, name, ci_type, status, environment, criticality
  FROM configuration_items WHERE tenant_id=1 ORDER BY ci_type, id;

\echo
\echo '=========== CI RELATIONSHIPS ==========='
SELECT r.id, src.name AS source, r.relationship_type, dst.name AS target,
       r.strength, r.impact_level, r.is_active
  FROM ci_relationships r
  JOIN configuration_items src ON src.id=r.source_ci_id
  JOIN configuration_items dst ON dst.id=r.target_ci_id
  WHERE r.tenant_id=1
  ORDER BY src.name, dst.name;

COMMIT;
