#!/bin/bash

# 清理已提交到 Git 仓库的目录（如 .gocache, .gomodcache 等）
# 使用方法: ./clean_committed_dirs.sh

set -e

echo "🚨 警告：此操作将重写 Git 历史，请确保已备份仓库！"
echo "⚠️  如果仓库已推送到远程，需要强制推送：git push --force --all"
echo ""
echo "按 Ctrl+C 取消，或按 Enter 继续..."
read

echo "开始清理已提交的目录..."

# 备份当前分支
echo "📦 创建备份分支..."
git branch backup-before-cleanup-$(date +%Y%m%d-%H%M%S) || true

# 要删除的目录列表
DIRS_TO_REMOVE=(
    "itsm-backend/.gocache"
    "itsm-backend/.gomodcache"
    "itsm-backend/.gopath"
    ".gocache"
    ".gomodcache"
    ".gopath"
)

# 检查是否安装了 git-filter-repo
if command -v git-filter-repo &> /dev/null; then
    echo "✅ 使用 git-filter-repo（推荐方法）..."
    
    # 使用 git-filter-repo 删除目录
    for dir in "${DIRS_TO_REMOVE[@]}"; do
        if git ls-tree -r --name-only HEAD | grep -q "^${dir}"; then
            echo "🗑️  删除: ${dir}"
            git filter-repo --path "${dir}" --invert-paths --force
        else
            echo "ℹ️  跳过: ${dir}（未在仓库中找到）"
        fi
    done
    
    echo "✅ git-filter-repo 清理完成！"
    
elif command -v git-filter-branch &> /dev/null; then
    echo "⚠️  使用 git-filter-branch（较慢，但兼容性好）..."
    
    # 使用 git-filter-branch 删除目录
    for dir in "${DIRS_TO_REMOVE[@]}"; do
        if git ls-tree -r --name-only HEAD | grep -q "^${dir}"; then
            echo "🗑️  删除: ${dir}"
            git filter-branch --force --index-filter \
                "git rm -r --cached --ignore-unmatch ${dir}" \
                --prune-empty --tag-name-filter cat -- --all
        else
            echo "ℹ️  跳过: ${dir}（未在仓库中找到）"
        fi
    done
    
    # 清理引用
    echo "🧹 清理引用..."
    rm -rf .git/refs/original/
    git reflog expire --expire=now --all
    git gc --prune=now --aggressive
    
    echo "✅ git-filter-branch 清理完成！"
    
else
    echo "❌ 错误：未找到 git-filter-repo 或 git-filter-branch"
    echo ""
    echo "请安装 git-filter-repo（推荐）："
    echo "  macOS: brew install git-filter-repo"
    echo "  Ubuntu: apt-get install git-filter-repo"
    echo "  pip: pip install git-filter-repo"
    exit 1
fi

echo ""
echo "✅ 清理完成！"
echo ""
echo "📊 检查仓库大小变化："
du -sh .git

echo ""
echo "📝 后续步骤："
echo "1. 检查清理结果: git log --all --oneline"
echo "2. 如果已推送到远程，需要强制推送:"
echo "   git push --force --all"
echo "   git push --force --tags"
echo "3. 通知团队成员重新克隆仓库或重置本地分支"
