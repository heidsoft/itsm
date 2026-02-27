#!/bin/bash
# 检查 GitHub Actions 执行状态
# 用于 OpenClaw cronjob

REPO="heidsoft/itsm"
API_URL="https://api.github.com/repos/${REPO}/actions/runs?per_page=5"

# 获取最近的 workflow runs
response=$(curl -s "$API_URL")

if [ $? -ne 0 ]; then
    echo "❌ 无法连接 GitHub API"
    exit 1
fi

# 解析最新的 3 个 runs
echo "## 📊 GitHub Actions 状态检查"
echo ""
echo "**仓库**: ${REPO}"
echo "**检查时间**: $(date '+%Y-%m-%d %H:%M:%S %Z')"
echo ""
echo "### 最近的构建"
echo ""

# 使用 jq 解析 JSON（如果可用）
if command -v jq &> /dev/null; then
    echo "$response" | jq -r '.workflow_runs[:3] | .[] | "- **\(.name)**: `\(.status)` - `\(.conclusion // "running"\)` [\(.head_branch)](\(.html_url))"'
else
    # 简单解析
    echo "$response" | grep -E '"name"|"status"|"conclusion"|"html_url"' | head -20
fi

echo ""
echo "### 状态说明"
echo "- ✅ `completed` + `success`: 构建成功"
echo "- ❌ `completed` + `failure`: 构建失败"
echo "- ⏳ `queued` / `in_progress`: 构建中"
echo ""
echo "---"
echo "_下次检查：15 分钟后_"
