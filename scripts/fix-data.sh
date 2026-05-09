#!/bin/bash
#
# ITSM 数据修复脚本
# 修复测试过程中发现的数据问题
#

set -e

DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-itsm}
DB_PASSWORD=${DB_PASSWORD:-itsm_password_2026}
DB_NAME=${DB_NAME:-itsm}

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=========================================="
echo "  ITSM 数据修复脚本"
echo "=========================================="
echo ""

# 检查是否有 psql
if ! command -v psql &> /dev/null; then
    echo -e "${YELLOW}psql 未安装，尝试使用 Docker 运行...${NC}"
    PSQL_CMD="docker exec -i itsm-postgres psql -U itsm -d itsm"
else
    PSQL_CMD="psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME"
fi

echo -e "${YELLOW}1. 发布知识库文章...${NC}"
$PSQL_CMD << 'EOF'
-- 发布所有草稿状态的知识库文章
UPDATE knowledge_articles 
SET is_published = true,
    updated_at = NOW()
WHERE is_published = false AND tenant_id = 1;

-- 显示更新结果
SELECT '知识库文章已发布' as result, COUNT(*) as count 
FROM knowledge_articles 
WHERE is_published = true AND tenant_id = 1;
EOF

echo -e "${YELLOW}2. 修复许可证到期日期...${NC}"
$PSQL_CMD << 'EOF'
-- 修复空白的到期日期
UPDATE asset_licenses
SET expiry_date = '2026-12-31 23:59:59',
    updated_at = NOW()
WHERE (expiry_date = '' OR expiry_date IS NULL) AND tenant_id = 1;

-- 显示更新结果
SELECT '许可证到期日期已修复' as result, COUNT(*) as count
FROM asset_licenses
WHERE expiry_date IS NOT NULL AND expiry_date != '' AND tenant_id = 1;
EOF

echo -e "${YELLOW}3. 创建示例配置项 (CMDB)...${NC}"
$PSQL_CMD << 'EOF'
-- 检查是否已有数据
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM configuration_items WHERE tenant_id = 1 LIMIT 1) THEN
        -- 插入示例服务器配置项 (使用正确的 ci_type_id)
        INSERT INTO configuration_items (name, ci_type, ci_type_id, status, environment, tenant_id, created_at, updated_at)
        VALUES 
        ('Web Server 01', '服务器', 1, 'active', 'production', 1, NOW(), NOW()),
        ('Web Server 02', '服务器', 1, 'active', 'production', 1, NOW(), NOW()),
        ('Database Server 01', '数据库', 2, 'active', 'production', 1, NOW(), NOW()),
        ('Load Balancer 01', '网络设备', 3, 'active', 'production', 1, NOW(), NOW()),
        ('Redis Cache 01', '存储', 4, 'active', 'production', 1, NOW(), NOW()),
        ('NFS Storage 01', '存储', 4, 'active', 'production', 1, NOW(), NOW()),
        ('API Gateway 01', '应用服务', 5, 'active', 'production', 1, NOW(), NOW()),
        ('Message Queue 01', '中间件', 6, 'active', 'production', 1, NOW(), NOW());
        
        RAISE NOTICE '已创建 8 个示例配置项';
    ELSE
        RAISE NOTICE '配置项已存在，跳过创建';
    END IF;
END $$;

-- 显示配置项数量
SELECT 'CMDB配置项' as item, COUNT(*) as count FROM configuration_items WHERE tenant_id = 1;
EOF

echo -e "${YELLOW}4. 验证 SLA 数据...${NC}"
$PSQL_CMD << 'EOF'
-- 检查 SLA 定义
SELECT 'SLA定义' as item, COUNT(*) as count FROM sla_definitions WHERE tenant_id = 1;

-- 检查 SLA 违规
SELECT 'SLA违规记录' as item, COUNT(*) as count FROM sla_violations WHERE tenant_id = 1;
EOF

echo ""
echo -e "${GREEN}=========================================="
echo "  数据修复完成!"
echo -e "==========================================${NC}"
echo ""
echo "请刷新页面查看修复结果:"
echo "  - http://localhost:3000/sla (SLA 合规率)"
echo "  - http://localhost:3000/cmdb (配置项)"
echo "  - http://localhost:3000/knowledge (知识库)"
echo "  - http://localhost:3000/licenses (许可证)"
echo ""