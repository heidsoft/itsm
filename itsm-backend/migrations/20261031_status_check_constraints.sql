-- 20261031_status_check_constraints.sql
--
-- 状态词表护栏：把「应用层枚举」落到数据库 CHECK 约束。
--
-- 背景（DB 审计 P2-5）：状态值有四个来源——common/dto 常量、各域状态机、
-- ent schema 默认值、数据库存量数据——彼此没有任何机制保证一致。实测：
--   * common.ProblemStatusNew = "new"，而问题域创建/状态机/报表全用 "open"；
--   * changes 域同时存在 pending（dto/API）与 submitted（service 状态机）两个
--     待审批写法；
--   * service_requests 的真实词表是 9 个审批/交付状态，ent 字段注释写的却是
--     另一套（submitted|approved|rejected|in_progress|completed|cancelled）。
-- 这类漂移不会报错：写入一个没人认识的状态，列表照常返回，只是报表把它归到
-- 「其他」、状态机把它当终态、前端显示空白。
--
-- 本迁移把权威词表（itsm-backend/common/status_vocabulary.go）落成 CHECK 约束，
-- 由 common/status_vocabulary_test.go 解析本文件的字面量与 Go 清单双向比对，
-- 任何一侧单独改动都会让测试变红。
--
-- 执行姿态（为什么是 NOT VALID + 守卫式 VALIDATE）：
--   * ADD CONSTRAINT ... NOT VALID 只做一次目录写入，不扫描存量行，秒级完成，
--     不阻塞热表读写；从这一刻起新写入即被拦截。
--   * 随后尝试 VALIDATE：存量数据若含词表外值，VALIDATE 会失败——此时**不让
--     整个迁移失败**（否则生产升级被历史脏数据卡死），而是保留 NOT VALID 并
--     RAISE WARNING，把「哪些表还有存量脏数据」如实暴露给运维。新写入仍然
--     受约束保护，存量数据需要单独清洗后再 VALIDATE。
--   * 幂等：约束已存在则跳过 ADD（不覆盖已验证状态），VALIDATE 重复执行是
--     廉价的 no-op；表/列不存在（fresh 库尚未建表）则跳过并 NOTICE。
--
-- 存量数据清洗口径（不在本迁移内自动执行）：
--   SELECT status, count(*) FROM <table> GROUP BY status;  -- 对照本文件字面量
--   changes.status 的 submitted 与 pending 语义相同，清洗时应统一为 pending
--   （API/DTO 词表），或按产品决策反向统一——二选一后从词表删除另一个。
--
-- 回滚：见 20261031_status_check_constraints_down.sql（discovery 按 *_down.sql 成对挂到 RollbackSQL）。

-- ============================================================================
-- 1. tickets.status
-- ============================================================================
DO $$
BEGIN
    IF to_regclass('public.tickets') IS NULL THEN
        RAISE NOTICE 'skip ck_tickets_status_vocabulary: table tickets missing';
        RETURN;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'tickets' AND column_name = 'status'
    ) THEN
        RAISE NOTICE 'skip ck_tickets_status_vocabulary: column tickets.status missing';
        RETURN;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ck_tickets_status_vocabulary'
          AND conrelid = 'public.tickets'::regclass
    ) THEN
        ALTER TABLE tickets ADD CONSTRAINT ck_tickets_status_vocabulary
            CHECK (status = ANY (ARRAY['new','open','assigned','in_progress','pending','resolved','closed','cancelled','approved','rejected']::text[]))
            NOT VALID;
    END IF;

    BEGIN
        ALTER TABLE tickets VALIDATE CONSTRAINT ck_tickets_status_vocabulary;
    EXCEPTION WHEN check_violation THEN
        RAISE WARNING 'ck_tickets_status_vocabulary 保持 NOT VALID：tickets.status 存在词表外存量值（新写入已被拦截，存量需清洗后重跑本迁移）: %', SQLERRM;
    END;
END $$;

-- ============================================================================
-- 2. incidents.status
-- ============================================================================
DO $$
BEGIN
    IF to_regclass('public.incidents') IS NULL THEN
        RAISE NOTICE 'skip ck_incidents_status_vocabulary: table incidents missing';
        RETURN;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'incidents' AND column_name = 'status'
    ) THEN
        RAISE NOTICE 'skip ck_incidents_status_vocabulary: column incidents.status missing';
        RETURN;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ck_incidents_status_vocabulary'
          AND conrelid = 'public.incidents'::regclass
    ) THEN
        ALTER TABLE incidents ADD CONSTRAINT ck_incidents_status_vocabulary
            CHECK (status = ANY (ARRAY['new','acknowledged','assigned','in_progress','triaged','escalated','on_hold','resolved','closed','cancelled']::text[]))
            NOT VALID;
    END IF;

    BEGIN
        ALTER TABLE incidents VALIDATE CONSTRAINT ck_incidents_status_vocabulary;
    EXCEPTION WHEN check_violation THEN
        RAISE WARNING 'ck_incidents_status_vocabulary 保持 NOT VALID：incidents.status 存在词表外存量值（新写入已被拦截，存量需清洗后重跑本迁移）: %', SQLERRM;
    END;
END $$;

-- ============================================================================
-- 3. problems.status
-- ============================================================================
DO $$
BEGIN
    IF to_regclass('public.problems') IS NULL THEN
        RAISE NOTICE 'skip ck_problems_status_vocabulary: table problems missing';
        RETURN;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'problems' AND column_name = 'status'
    ) THEN
        RAISE NOTICE 'skip ck_problems_status_vocabulary: column problems.status missing';
        RETURN;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ck_problems_status_vocabulary'
          AND conrelid = 'public.problems'::regclass
    ) THEN
        ALTER TABLE problems ADD CONSTRAINT ck_problems_status_vocabulary
            CHECK (status = ANY (ARRAY['open','investigating','identified','in_progress','resolved','closed']::text[]))
            NOT VALID;
    END IF;

    BEGIN
        ALTER TABLE problems VALIDATE CONSTRAINT ck_problems_status_vocabulary;
    EXCEPTION WHEN check_violation THEN
        RAISE WARNING 'ck_problems_status_vocabulary 保持 NOT VALID：problems.status 存在词表外存量值（新写入已被拦截，存量需清洗后重跑本迁移）: %', SQLERRM;
    END;
END $$;

-- ============================================================================
-- 4. changes.status
--    pending 与 submitted 是同一语义的两个历史写法，两者都必须接受，
--    否则存量变更无法读取、也无法迁移到规范词表。
-- ============================================================================
DO $$
BEGIN
    IF to_regclass('public.changes') IS NULL THEN
        RAISE NOTICE 'skip ck_changes_status_vocabulary: table changes missing';
        RETURN;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'changes' AND column_name = 'status'
    ) THEN
        RAISE NOTICE 'skip ck_changes_status_vocabulary: column changes.status missing';
        RETURN;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ck_changes_status_vocabulary'
          AND conrelid = 'public.changes'::regclass
    ) THEN
        ALTER TABLE changes ADD CONSTRAINT ck_changes_status_vocabulary
            CHECK (status = ANY (ARRAY['draft','pending','submitted','approved','rejected','scheduled','in_progress','completed','failed','rolled_back','cancelled','closed']::text[]))
            NOT VALID;
    END IF;

    BEGIN
        ALTER TABLE changes VALIDATE CONSTRAINT ck_changes_status_vocabulary;
    EXCEPTION WHEN check_violation THEN
        RAISE WARNING 'ck_changes_status_vocabulary 保持 NOT VALID：changes.status 存在词表外存量值（新写入已被拦截，存量需清洗后重跑本迁移）: %', SQLERRM;
    END;
END $$;

-- ============================================================================
-- 5. service_requests.status
-- ============================================================================
DO $$
BEGIN
    IF to_regclass('public.service_requests') IS NULL THEN
        RAISE NOTICE 'skip ck_service_requests_status_vocabulary: table service_requests missing';
        RETURN;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'service_requests' AND column_name = 'status'
    ) THEN
        RAISE NOTICE 'skip ck_service_requests_status_vocabulary: column service_requests.status missing';
        RETURN;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ck_service_requests_status_vocabulary'
          AND conrelid = 'public.service_requests'::regclass
    ) THEN
        ALTER TABLE service_requests ADD CONSTRAINT ck_service_requests_status_vocabulary
            CHECK (status = ANY (ARRAY['submitted','manager_approved','it_approved','security_approved','rejected','provisioning','delivered','failed','cancelled']::text[]))
            NOT VALID;
    END IF;

    BEGIN
        ALTER TABLE service_requests VALIDATE CONSTRAINT ck_service_requests_status_vocabulary;
    EXCEPTION WHEN check_violation THEN
        RAISE WARNING 'ck_service_requests_status_vocabulary 保持 NOT VALID：service_requests.status 存在词表外存量值（新写入已被拦截，存量需清洗后重跑本迁移）: %', SQLERRM;
    END;
END $$;

-- ============================================================================
-- 6. knowledge_articles.review_status
-- ============================================================================
DO $$
BEGIN
    IF to_regclass('public.knowledge_articles') IS NULL THEN
        RAISE NOTICE 'skip ck_knowledge_articles_review_status_vocabulary: table knowledge_articles missing';
        RETURN;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'knowledge_articles' AND column_name = 'review_status'
    ) THEN
        RAISE NOTICE 'skip ck_knowledge_articles_review_status_vocabulary: column knowledge_articles.review_status missing';
        RETURN;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ck_knowledge_articles_review_status_vocabulary'
          AND conrelid = 'public.knowledge_articles'::regclass
    ) THEN
        ALTER TABLE knowledge_articles ADD CONSTRAINT ck_knowledge_articles_review_status_vocabulary
            CHECK (review_status = ANY (ARRAY['draft','under_review','approved','rejected']::text[]))
            NOT VALID;
    END IF;

    BEGIN
        ALTER TABLE knowledge_articles VALIDATE CONSTRAINT ck_knowledge_articles_review_status_vocabulary;
    EXCEPTION WHEN check_violation THEN
        RAISE WARNING 'ck_knowledge_articles_review_status_vocabulary 保持 NOT VALID：knowledge_articles.review_status 存在词表外存量值（新写入已被拦截，存量需清洗后重跑本迁移）: %', SQLERRM;
    END;
END $$;
