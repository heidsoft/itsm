-- 测试数据初始化脚本
-- 用于补充知识库和问题管理的测试数据

-- ============================================
-- 知识库文章测试数据
-- ============================================

INSERT INTO knowledge_articles (title, content, category, tags, author_id, tenant_id, is_published, view_count, like_count, created_at, updated_at)
VALUES 
  ('IT服务台使用指南', '<h2>IT服务台使用指南</h2><p>本文档介绍如何使用IT服务台系统。</p>', '使用指南', '指南,IT服务台', 1, 1, true, 125, 8, NOW(), NOW()),
  ('如何提交工单', '<h2>如何提交工单</h2><p>工单提交流程说明。</p>', '使用指南', '工单,提交', 1, 1, true, 98, 5, NOW(), NOW()),
  ('密码重置流程', '<h2>密码重置流程</h2><p>如果忘记密码，请按以下步骤操作。</p>', '安全', '密码,安全', 1, 1, true, 156, 12, NOW(), NOW()),
  ('VPN连接配置', '<h2>VPN连接配置</h2><p>远程办公VPN配置说明。</p>', '网络', 'VPN,远程', 1, 1, true, 87, 6, NOW(), NOW()),
  ('邮件系统故障排除', '<h2>邮件系统故障排除</h2><p>常见邮件问题及解决方案。</p>', '故障排除', '邮件,故障', 1, 1, false, 45, 3, NOW(), NOW()),
  ('打印机配置指南', '<h2>打印机配置指南</h2><p>网络打印机配置步骤。</p>', '硬件', '打印机,配置', 1, 1, false, 32, 2, NOW(), NOW()),
  ('云资源申请流程', '<h2>云资源申请流程</h2><p>如何申请云服务器和存储。</p>', '云资源', '云,资源申请', 1, 1, true, 78, 4, NOW(), NOW()),
  ('数据库备份策略', '<h2>数据库备份策略</h2><p>数据库备份和恢复指南。</p>', '数据库', '数据库,备份', 1, 1, true, 112, 9, NOW(), NOW()),
  ('开发环境搭建', '<h2>开发环境搭建</h2><p>本地开发环境配置指南。</p>', '研发效能', '开发,环境', 1, 1, true, 203, 15, NOW(), NOW()),
  ('安全漏洞修复指南', '<h2>安全漏洞修复指南</h2><p>常见安全漏洞修复方法。</p>', '安全', '安全,漏洞', 1, 1, false, 28, 1, NOW(), NOW())
ON CONFLICT DO NOTHING;

-- ============================================
-- 问题管理测试数据
-- ============================================

INSERT INTO problems (title, description, priority, status, category, tenant_id, created_by, assigned_to, created_at, updated_at)
VALUES 
  ('服务器响应缓慢', '生产环境服务器响应时间超过5秒', 'high', 'open', '性能问题', 1, 1, 1, NOW(), NOW()),
  ('数据库连接池耗尽', '数据库连接池频繁耗尽导致服务中断', 'critical', 'in_progress', '数据库', 1, 1, 1, NOW(), NOW()),
  ('存储空间不足', '日志服务器磁盘空间不足', 'medium', 'open', '存储', 1, 1, NULL, NOW(), NOW()),
  ('网络延迟增加', '跨区域网络延迟明显增加', 'high', 'resolved', '网络', 1, 1, 1, NOW(), NOW()),
  ('认证服务异常', 'OAuth认证服务偶发性失败', 'critical', 'in_progress', '安全', 1, 1, 1, NOW(), NOW()),
  ('备份任务失败', '每日备份任务持续失败', 'medium', 'open', '运维', 1, 1, NULL, NOW(), NOW()),
  ('API限流问题', '部分API接口频繁触发限流', 'low', 'closed', '性能问题', 1, 1, 1, NOW(), NOW()),
  ('内存泄漏排查', '应用服务器存在内存泄漏', 'high', 'in_progress', '性能问题', 1, 1, 1, NOW(), NOW())
ON CONFLICT DO NOTHING;

-- ============================================
-- 验证数据插入
-- ============================================

SELECT 'Knowledge Articles:' as info, COUNT(*) as count FROM knowledge_articles WHERE tenant_id = 1
UNION ALL
SELECT 'Published Articles:', COUNT(*) FROM knowledge_articles WHERE tenant_id = 1 AND is_published = true
UNION ALL
SELECT 'Draft Articles:', COUNT(*) FROM knowledge_articles WHERE tenant_id = 1 AND is_published = false
UNION ALL
SELECT 'Problems:', COUNT(*) FROM problems WHERE tenant_id = 1
UNION ALL
SELECT 'Open Problems:', COUNT(*) FROM problems WHERE tenant_id = 1 AND status = 'open';
