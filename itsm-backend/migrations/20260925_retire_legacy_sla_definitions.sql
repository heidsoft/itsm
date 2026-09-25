-- 20260925_retire_legacy_sla_definitions.sql
--
-- 停用两代旧命名的受管 SLA 定义（决策 2026-09-25：改名/停用/保留 → 停用）。
--
-- 背景：SLA 定义清单命名演进过两代（SLA-P0-紧急…SLA-变更，[Template] 事件/变更…），
-- 前滚 reconcile 只按清单补齐新命名条目，不会删改旧条目，于是存量库出现
-- 「旧代孤儿 + 新代正式」并存。旧代仍被工单/SLA 违规记录引用（本地栈实测 17 张工单、
-- 26 条违规挂在旧代上），不能删除；改名会与已创建的新代撞名（verify 用 Only()，
-- 同名双行直接校验失败），且 SLA-变更 一对二映射语义靠猜。
--
-- 因此只把封闭清单内的旧代条目置 is_active=false：
--   - 不删行：历史工单/违规引用继续可解析，保留原 SLA 条款；
--   - 不改名：新代条目仍是唯一正式命名；
--   - 新工单不再可选旧代；幂等（只翻 active=true 的行）。
--
-- 执行顺序：可与产品基线前滚（initialize -action apply / provision_tenant）任意先后，
-- 两者互不依赖。只匹配下列精确名称，客户自建 SLA 一概不动。
--
-- 已知边界：同名的客户自建行会被一并停用（封闭清单均为产品模板历史命名，重名概率极低）。

DO $$
DECLARE
    flipped INTEGER;
BEGIN
    UPDATE sla_definitions
    SET is_active = false,
        updated_at = now()
    WHERE is_active = true
      AND name IN (
        'SLA-P0-紧急',
        'SLA-P1-高',
        'SLA-P2-中',
        'SLA-P3-低',
        'SLA-服务请求',
        'SLA-变更',
        '[Template] 事件 P1 紧急',
        '[Template] 事件 P2 高',
        '[Template] 事件 P3 中',
        '[Template] 变更 - 普通',
        '[Template] 变更 - 紧急'
      );
    GET DIAGNOSTICS flipped = ROW_COUNT;
    RAISE NOTICE 'legacy SLA definitions retired this run: %', flipped;
END $$;
