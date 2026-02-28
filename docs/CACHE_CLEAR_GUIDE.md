# GitHub Actions 缓存清除指南

**问题**: CI 构建使用缓存的旧依赖，导致新添加的 dayjs 依赖未生效  
**日期**: 2026-02-28  
**状态**: ✅ 已解决

---

## 🔍 问题根因

### 缓存机制
GitHub Actions 会缓存 `node_modules` 以提高构建速度，但当：
- `package.json` 或 `package-lock.json` 变更
- 添加了新依赖（如 dayjs）
- 依赖版本冲突

缓存可能不会自动失效，导致使用旧依赖。

---

## ✅ 解决方案

### 方案 1: 手动清除缓存（推荐）

#### GitHub Web 界面
1. 访问：https://github.com/heidsoft/itsm/actions/caches
2. 点击 **"Delete all caches"**
3. 确认删除
4. 重新触发构建

#### 或使用 API
```bash
# 需要 GitHub CLI 或 curl
gh api \
  --method DELETE \
  /repos/heidsoft/itsm/actions/caches
```

### 方案 2: 使用缓存清除工作流

已创建 `clear-cache.yml`：

```yaml
name: Clear CI Cache

on:
  workflow_dispatch:  # 手动触发

jobs:
  clear-cache:
    runs-on: ubuntu-latest
    steps:
      - name: Clear npm cache
        run: |
          echo "Clearing npm cache..."
          npm cache clean --force || true
          echo "Cache cleared!"
```

**使用方法**:
1. 访问：https://github.com/heidsoft/itsm/actions/workflows/clear-cache.yml
2. 点击 **"Run workflow"**
3. 选择分支（main）
4. 点击 **"Run workflow"**

### 方案 3: 修改缓存键

在 CI 配置中添加版本号强制刷新：

```yaml
- name: Setup Node.js
  uses: actions/setup-node@v4
  with:
    node-version: '22'
    cache: 'npm'
    cache-dependency-path: itsm-frontend/package-lock.json
    
- name: Clear npm cache
  run: npm cache clean --force
  working-directory: ./itsm-frontend
```

---

## 🔧 本次修复

### 1. 添加 --legacy-peer-deps

**文件**: `.github/workflows/frontend-ci.yml`

```yaml
- name: Install dependencies
  working-directory: ./itsm-frontend
  run: npm ci --legacy-peer-deps  # ✅ 添加此参数
```

**原因**:
- antd 6.x 与其他包有 peer dependency 冲突
- `--legacy-peer-deps` 允许忽略这些冲突
- 与本地开发保持一致

### 2. 添加 dayjs 依赖

**文件**: `itsm-frontend/package.json`

```json
{
  "dependencies": {
    "dayjs": "^1.11.13"  // ✅ 新增
  }
}
```

### 3. 添加类型声明

**文件**: `itsm-frontend/src/types/dayjs.d.ts`

```typescript
import dayjs from 'dayjs';

declare module 'dayjs' {
  interface Dayjs {
    format(template?: string): string;
    // ... 其他方法
  }
}
```

---

## 📊 验证步骤

### 1. 清除缓存
```bash
# 访问 GitHub Actions
https://github.com/heidsoft/itsm/actions/caches

# 删除所有缓存
```

### 2. 触发新构建
```bash
# 推送空提交触发构建
git commit --allow-empty -m "ci: trigger rebuild with fresh cache"
git push origin main
```

### 3. 验证依赖
```yaml
# 在 CI 日志中查找
- name: Check installed packages
  run: npm list dayjs
  working-directory: ./itsm-frontend
```

### 4. 验证类型
```yaml
# Type Check 应该通过
- name: Run type check
  run: npm run type-check
  working-directory: ./itsm-frontend
```

---

## 🎯 最佳实践

### 1. 缓存策略
```yaml
# ✅ 推荐：使用 package-lock.json hash 作为缓存键
- name: Setup Node.js
  uses: actions/setup-node@v4
  with:
    cache: 'npm'
    cache-dependency-path: itsm-frontend/package-lock.json
```

### 2. 缓存失效条件
- `package-lock.json` 变更 → 自动失效
- 手动删除缓存 → 立即失效
- 修改缓存键 → 强制失效

### 3. 依赖管理
```bash
# ✅ 添加新依赖
npm install dayjs --save

# ✅ 更新依赖
npm update dayjs

# ✅ 修复依赖
npm install --legacy-peer-deps
```

### 4. CI 配置
```yaml
# ✅ 推荐：添加缓存清除步骤（可选）
- name: Clear npm cache
  run: npm cache clean --force
  if: github.event_name == 'workflow_dispatch'
  
- name: Install dependencies
  run: npm ci --legacy-peer-deps
```

---

## 📝 故障排查

### 问题 1: 构建仍然失败
**检查**:
```bash
# 查看使用的依赖版本
npm list dayjs

# 查看缓存状态
npm cache verify

# 强制清除
npm cache clean --force
```

### 问题 2: 缓存未清除
**解决**:
1. 手动删除：https://github.com/heidsoft/itsm/actions/caches
2. 修改缓存键：添加版本号后缀
3. 使用 `npm ci --force`

### 问题 3: 依赖冲突
**解决**:
```bash
# 使用 --legacy-peer-deps
npm ci --legacy-peer-deps

# 或使用 --force（不推荐）
npm ci --force
```

---

## 🔗 相关资源

- [GitHub Actions Caching](https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows)
- [npm cache](https://docs.npmjs.com/cli/commands/npm-cache)
- [Clear GitHub Actions Cache](https://github.com/actions/cache/issues/643)

---

**维护者**: ITSM Team  
**最后更新**: 2026-02-28
