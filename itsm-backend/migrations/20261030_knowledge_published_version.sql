-- ============================================================
-- 知识文章：发布版本指针（线上内容的唯一事实源）
-- 2026-10-30 / knowledge 域 pilotReason 第一项修复
--
-- 背景：发布此前只是把 is_published 翻 true，版本表存的是「更新前快照」，
-- 发布时刻的正文在版本表没有对应行——「线上是哪个版本」无解。
-- 语义：published_version 指向发布时落的快照行；nil=从未发布或已撤回。
-- 撤回与恢复（恢复强制回 draft）清空指针。
--
-- 幂等：列存在则跳过；无数据回填（存量文章无法可靠推断发布时刻的正文，
-- 指针保持 nil，语义如实为「无发布版本记录」，下一次发布起开始有指针）。
-- ============================================================

DO $$
DECLARE
    col_exists boolean;
BEGIN
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'knowledge_articles'
          AND column_name = 'published_version'
    ) INTO col_exists;

    IF NOT col_exists THEN
        ALTER TABLE knowledge_articles
            ADD COLUMN published_version INTEGER NULL;

        COMMENT ON COLUMN knowledge_articles.published_version IS
            '线上版本号；空=从未发布或已撤回。发布时先为上线正文落版本快照再指过去';
    END IF;
END $$;
