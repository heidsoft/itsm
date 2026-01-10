#!/bin/bash

echo "🚨 警告：此操作将重写 Git 历史，请确保已备份仓库！"
echo "按 Ctrl+C 取消，或按 Enter 继续..."
read

echo "开始清理大文件..."

# 备份当前分支
git branch backup-before-cleanup

# 清理 itsm-backend 可执行文件
echo "清理 itsm-backend 可执行文件..."
git filter-branch --force --index-filter \
  'git rm --cached --ignore-unmatch itsm-backend/itsm-backend*' \
  --prune-empty --tag-name-filter cat -- --all

# 清理 node_modules 中的大文件
echo "清理 node_modules 大文件..."
git filter-branch --force --index-filter \
  'git rm -r --cached --ignore-unmatch itsm-frontend/node_modules' \
  --prune-empty --tag-name-filter cat -- --all

# 清理 .jest-cache 中的大文件
echo "清理 .jest-cache 大文件..."
git filter-branch --force --index-filter \
  'git rm -r --cached --ignore-unmatch itsm-frontend/.jest-cache' \
  --prune-empty --tag-name-filter cat -- --all

# 清理 .next 中的大文件
echo "清理 .next 大文件..."
git filter-branch --force --index-filter \
  'git rm -r --cached --ignore-unmatch itsm-frontend/.next' \
  --prune-empty --tag-name-filter cat -- --all


# 清理 itsm-backend/.gocache 中的大文件
echo "清理 itsm-backend/.gocache 大文件..."
git filter-branch --force --index-filter \
  'git rm -r --cached --ignore-unmatch itsm-backend/.gocache' \
  --prune-empty --tag-name-filter cat -- --all


# 清理 itsm-backend/.gomodcache 中的大文件
echo "清理 itsm-backend/.gomodcache 大文件..."
git filter-branch --force --index-filter \
  'git rm -r --cached --ignore-unmatch itsm-backend/.gomodcache' \
  --prune-empty --tag-name-filter cat -- --all

# 清理 itsm-backend/.gopath 中的大文件
echo "清理 itsm-backend/.gopath 大文件..."
git filter-branch --force --index-filter \
  'git rm -r --cached --ignore-unmatch itsm-backend/.gopath' \
  --prune-empty --tag-name-filter cat -- --all



# 清理引用
echo "清理引用..."
rm -rf .git/refs/original/
git reflog expire --expire=now --all
git gc --prune=now --aggressive

echo "✅ 清理完成！"
echo "📊 检查仓库大小变化："
du -sh .git

