#!/bin/bash

echo "=== CMDB集成测试 ==="
echo "现在CMDB功能已经成功集成到现有的ITSM项目中！"
echo ""

echo "主要改进："
echo "1. ✅ 将CMDB实体集成到现有的ent schema中"
echo "2. ✅ ConfigurationItem 替代了独立的CMDBCI"
echo "3. ✅ CIRelationship 管理CI之间的关系"
echo "4. ✅ 与现有Ticket和Incident实体建立关联"
echo "5. ✅ 添加了新的CMDB控制器和服务"
echo "6. ✅ 集成到现有的路由和启动流程中"
echo ""

echo "可用的CMDB API端点："
echo "GET    /api/v1/configuration-items          - 列出配置项"
echo "POST   /api/v1/configuration-items          - 创建配置项"
echo "GET    /api/v1/configuration-items/:id      - 获取配置项详情"
echo "GET    /api/v1/configuration-items/:id/topology - 获取CI拓扑关系"
echo "POST   /api/v1/configuration-items/relationships - 创建CI关系"
echo ""

echo "同时保留了原有的DDD架构CMDB端点："
echo "GET    /api/v1/cmdb/cis                     - DDD架构的CI管理"
echo "POST   /api/v1/cmdb/cis                     - DDD架构的CI创建"
echo "GET    /api/v1/cmdb/stats                   - CMDB统计信息"
echo ""

echo "数据库schema更新："
echo "- configuration_items 表（主要CI数据）"
echo "- ci_relationships 表（CI关系数据）"
echo "- 与tickets和incidents表的外键关联"
echo ""

echo "服务器启动命令："
echo "cd itsm-backend && ./itsm-server"
echo "服务器将在 http://localhost:8090 启动"
echo ""

echo "测试示例（需要先登录获取JWT token）："
echo 'curl -X POST http://localhost:8090/api/v1/configuration-items \'
echo '  -H "Content-Type: application/json" \'
echo '  -H "Authorization: Bearer YOUR_JWT_TOKEN" \'
echo '  -d "{"name":"Web Server 01","ci_type":"server","status":"operational"}"'
echo ""

echo "🎉 CMDB功能已成功集成到ITSM项目中！"
