-- 20261031_status_check_constraints_down.sql
--
-- 回滚 20261031_status_check_constraints.sql：逐表删除状态词表 CHECK 约束。
--
-- 注意：删除约束会让「写入未知状态」重新变成静默成功。回滚只应用于
-- 「约束误拦了合法业务写入」的应急场景；如果是词表本身要变更，正确做法是
-- 新增一个迁移改写约束，而不是回滚后长期缺失护栏。
--
-- 幂等：约束不存在则跳过。

DO $$
BEGIN
    IF to_regclass('public.tickets') IS NOT NULL THEN
        ALTER TABLE tickets DROP CONSTRAINT IF EXISTS ck_tickets_status_vocabulary;
    END IF;
    IF to_regclass('public.incidents') IS NOT NULL THEN
        ALTER TABLE incidents DROP CONSTRAINT IF EXISTS ck_incidents_status_vocabulary;
    END IF;
    IF to_regclass('public.problems') IS NOT NULL THEN
        ALTER TABLE problems DROP CONSTRAINT IF EXISTS ck_problems_status_vocabulary;
    END IF;
    IF to_regclass('public.changes') IS NOT NULL THEN
        ALTER TABLE changes DROP CONSTRAINT IF EXISTS ck_changes_status_vocabulary;
    END IF;
    IF to_regclass('public.service_requests') IS NOT NULL THEN
        ALTER TABLE service_requests DROP CONSTRAINT IF EXISTS ck_service_requests_status_vocabulary;
    END IF;
    IF to_regclass('public.knowledge_articles') IS NOT NULL THEN
        ALTER TABLE knowledge_articles DROP CONSTRAINT IF EXISTS ck_knowledge_articles_review_status_vocabulary;
    END IF;
END $$;
