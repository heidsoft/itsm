#!/bin/bash

echo "=== CMDB完整功能测试 ==="
echo "PostgreSQL@14 已启动，开始测试..."
echo ""

# 启动ITSM服务器
cd /Users/heidsoft/Downloads/research/itsm/itsm-backend
echo "启动ITSM服务器..."
./itsm-server &
SERVER_PID=$!
sleep 8

echo "1. 测试健康检查..."
curl -s http://localhost:8090/api/v1/health
echo ""

echo "2. 登录获取JWT token..."
TOKEN=$(curl -s -X POST http://localhost:8090/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | \
  jq -r '.data.access_token // empty')

if [ -n "$TOKEN" ]; then
  echo "✅ 登录成功，获得token"
  
  echo "3. 创建配置项..."
  CI_RESPONSE=$(curl -s -X POST http://localhost:8090/api/v1/configuration-items \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{
      "name": "Web Server 01",
      "ci_type": "server",
      "status": "operational",
      "environment": "production",
      "criticality": "high",
      "location": "Data Center A"
    }')
  
  echo "CI创建响应: $CI_RESPONSE"
  
  echo "4. 查询配置项列表..."
  curl -s -H "Authorization: Bearer $TOKEN" \
    http://localhost:8090/api/v1/configuration-items
  echo ""
  
  echo "5. 测试DDD架构的CMDB API..."
  curl -s -H "Authorization: Bearer $TOKEN" \
    http://localhost:8090/api/v1/cmdb/stats
  echo ""
  
else
  echo "❌ 登录失败"
fi

echo ""
echo "停止服务器..."
kill $SERVER_PID 2>/dev/null
echo "✅ 测试完成！"
