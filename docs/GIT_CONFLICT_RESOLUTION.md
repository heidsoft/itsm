# Git 合并冲突解决指南

## 常用命令

### 1. 识别冲突
```bash
git status                    # 查看冲突状态
git diff --name-only --diff-filter=U  # 只列出冲突文件
```

### 2. 查看冲突内容
文件中的冲突标记：
```
<<<<<<< HEAD
本地版本（当前分支）
=======
远程版本（合并的分支）
>>>>>>> commit-hash
```

### 3. 解决冲突

**手动编辑**：删除标记，保留/合并代码

**使用工具**：
```bash
git mergetool    # 启动合并工具
```

**保留某一侧**：
```bash
git checkout --ours file.go   # 保留本地
git checkout --theirs file.go # 保留远程
```

### 4. 完成合并
```bash
git add resolved-file.go
git commit -m "Merge remote-tracking branch 'origin/main' into main"
```

## 解决导入冲突的思路

1. 分析两侧代码使用的导入
2. 合并所有需要的导入
3. 移除重复导入
4. 确保导入顺序符合 Go 规范（标准库 → 第三方 → 项目内部）

## 最佳实践

- 冲突后先理解代码逻辑
- 解决后运行测试确保功能正常
- 提交信息说明解决方式
