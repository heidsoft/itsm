# NPM 缓存错误修复报告

**日期**: 2026-02-28  
**错误**: `Some specified paths were not resolved, unable to cache dependencies`  
**状态**: 🔄 修复中

---

## 🔍 错误详情

### 错误信息
```
Run actions/setup-node@v4
Found in cache @ /opt/hostedtoolcache/node/22.22.0/x64

Environment details
node: v22.22.0
npm: 10.9.4
yarn: 1.22.22

/home/runner/.npm
Error: Some specified paths were not resolved, unable to cache dependencies.
```

### 根因分析
- GitHub Actions 的 `cache: 'npm'` 配置与当前环境不兼容
- npm 缓存路径 `/home/runner/.npm` 无法正确解析
- 导致依赖安装失败

---

## ✅ 已实施的修复

### 修复方案：移除缓存配置

**文件**: `.github/workflows/frontend-ci.yml`

**Before**:
```yaml
- name: Setup Node.js
  uses: actions/setup-node@v4
  with:
    node-version: '22'
    cache: 'npm'  # ❌ 导致错误
    cache-dependency-path: itsm-frontend/package-lock.json
```

**After**:
```yaml
- name: Setup Node.js
  uses: actions/setup-node@v4
  with:
    node-version: '22'
    # ✅ 移除缓存配置，让 GitHub 自动处理

- name: Clear npm cache
  run: npm cache clean --force
  working-directory: ./itsm-frontend
  continue-on-error: true
```

### 修改范围
- ✅ frontend-ci.yml: 移除 5 处缓存配置
- ✅ 保留 `npm cache clean --force` 步骤
- ✅ 保持 Node 22 版本配置

---

## 📊 当前状态

### 最新提交
```
30c9aca - fix: remove npm cache config from frontend CI (最新)
3211035 - docs: document Node.js 22 configuration
e5bd0f8 - fix: add Node.js 22 engine requirement
```

### 构建状态
- 🔄 30c9aca - ITSM Frontend CI: 等待验证
- Node: 22.22.0 ✅
- npm: 10.9.4 ✅

---

## 🎯 预期效果

### 修复后
- ✅ 不再出现缓存路径错误
- ✅ 依赖正常安装
- ✅ Type Check 通过（dayjs 已修复）
- ✅ Build 成功

### 性能影响
- 移除缓存后，首次构建会稍慢（~1-2 分钟）
- 但 GitHub 会自动处理缓存，无需手动配置
- 后续构建会自动复用 node_modules

---

## 🔧 备选方案

### 如果仍然失败

**方案 1: 使用 npm install 替代 npm ci**
```yaml
- name: Install dependencies
  run: npm install
  working-directory: ./itsm-frontend
```

**方案 2: 添加 npm 配置**
```yaml
- name: Configure npm
  run: |
    npm config set cache /tmp/npm-cache
    mkdir -p /tmp/npm-cache
  working-directory: ./itsm-frontend
```

**方案 3: 完全禁用缓存**
```yaml
- name: Setup Node.js
  uses: actions/setup-node@v4
  with:
    node-version: '22'
    # 不添加任何缓存配置
```

---

## 📝 技术总结

### 学到的经验
1. **不要手动配置 npm 缓存** - GitHub Actions 会自动处理
2. **使用 `npm cache clean --force`** - 确保干净的依赖安装
3. **保持配置简单** - 越少配置越好

### 最佳实践
```yaml
# ✅ 推荐配置
- name: Setup Node.js
  uses: actions/setup-node@v4
  with:
    node-version: '22'

- name: Clear npm cache
  run: npm cache clean --force
  continue-on-error: true

- name: Install dependencies
  run: npm ci
  working-directory: ./itsm-frontend
```

---

## 🔗 相关资源

- [GitHub Actions Caching](https://docs.github.com/en/actions/using-workflows/caching-dependencies-to-speed-up-workflows)
- [setup-node Action](https://github.com/actions/setup-node)
- [npm Cache Issues](https://github.com/npm/cli/issues)

---

**维护者**: ITSM Team  
**修复时间**: 2026-02-28 14:00 CST
