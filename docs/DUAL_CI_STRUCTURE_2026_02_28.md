# 双 CI 配置结构说明

**日期**: 2026-02-28  
**状态**: ✅ 已统一  
**分支**: main

---

## 📊 项目结构

```
itsm/
├── .github/workflows/          # 根目录 CI 配置
│   ├── frontend-ci.yml         # 前端 CI（完整）
│   ├── frontend-tests.yml      # 前端测试
│   ├── backend-ci.yml          # 后端 CI
│   ├── release.yml             # 发布流程
│   ├── automated-tests.yml     # 自动化测试
│   └── clear-cache.yml         # 缓存清除
│
├── itsm-frontend/              # 前端目录
│   ├── .github/workflows/
│   │   └── ci.yml              # 前端 CI（与 frontend-ci.yml 同步）
│   ├── package.json
│   └── src/
│
└── itsm-backend/               # 后端目录
    ├── go.mod
    └── main.go
```

---

## 🎯 CI 配置说明

### 1. 根目录 CI (`itsm/.github/workflows/`)

**用途**: 完整的 CI/CD 流程，包含前后端

**工作流**:
- `frontend-ci.yml` - 前端完整 CI（lint, type-check, tests, build）
- `frontend-tests.yml` - 前端专项测试
- `backend-ci.yml` - 后端 CI（build, test）
- `release.yml` - 发布流程
- `automated-tests.yml` - 自动化测试
- `clear-cache.yml` - 手动缓存清除

**特点**:
- ✅ 统一的 Node 22 配置
- ✅ 统一的 npm 配置（无缓存）
- ✅ 完整的产物上传
- ✅ 超时保护

### 2. 前端目录 CI (`itsm/itsm-frontend/.github/workflows/`)

**用途**: 前端独立的 CI 流程

**工作流**:
- `ci.yml` - 前端 CI（与 frontend-ci.yml 同步）
- `test-performance.yml` - 性能测试

**特点**:
- ✅ 与根目录配置一致
- ✅ 可以独立运行
- ✅ 支持 PR 验证

---

## ✅ 已统一的配置

### Node.js 版本
```yaml
# 所有 CI 配置
- name: Setup Node.js
  uses: actions/setup-node@v4
  with:
    node-version: '22'  # ✅ 统一使用 Node 22
```

### 依赖安装
```yaml
# 所有 CI 配置
- name: Install dependencies
  run: npm ci  # ✅ 不使用 --legacy-peer-deps
```

### 缓存处理
```yaml
# 所有 CI 配置
- name: Clear npm cache
  run: npm cache clean --force  # ✅ 清除缓存
  continue-on-error: true
```

**注意**: 移除了 `cache: 'npm'` 配置，避免缓存路径错误。

---

## 🔧 配置对比

### frontend-ci.yml vs ci.yml

| 配置项 | frontend-ci.yml | ci.yml | 状态 |
|--------|----------------|--------|------|
| Node 版本 | 22 | 22 | ✅ 统一 |
| npm 缓存 | 无 | 无 | ✅ 统一 |
| 缓存清除 | ✅ | ✅ | ✅ 统一 |
| Lint | ✅ | ✅ | ✅ 统一 |
| Type Check | ✅ | ✅ | ✅ 统一 |
| Unit Tests | ✅ | ✅ | ✅ 统一 |
| Integration Tests | ✅ | ✅ | ✅ 统一 |
| Build | ✅ | ✅ | ✅ 统一 |
| 产物上传 | ✅ | ✅ | ✅ 统一 |
| 超时保护 | ✅ | ✅ | ✅ 统一 |

---

## 📝 使用指南

### 触发前端 CI

**方式 1: 推送到 main 分支**
```bash
git push origin main
# 自动触发：frontend-ci.yml
```

**方式 2: 创建 PR**
```bash
git checkout -b feature/new-feature
git push origin feature/new-feature
# 自动触发：ci.yml (PR 验证)
```

**方式 3: 手动触发**
```
GitHub Actions → ITSM Frontend CI → Run workflow
```

### 触发后端 CI

```bash
git push origin main
# 自动触发：backend-ci.yml
```

### 触发完整发布

```bash
git tag v1.0.1
git push origin v1.0.1
# 自动触发：release.yml
```

---

## 🎯 最佳实践

### 1. 配置同步
- ✅ 保持 `frontend-ci.yml` 和 `ci.yml` 配置一致
- ✅ 使用相同的 Node 版本
- ✅ 使用相同的 npm 配置

### 2. 缓存管理
- ✅ 每次构建前清除 npm 缓存
- ✅ 不手动配置缓存键
- ✅ 让 GitHub 自动处理

### 3. 产物管理
- ✅ 上传所有测试结果
- ✅ 上传构建产物
- ✅ 设置合理的保留时间（3-7 天）

### 4. 错误处理
- ✅ 添加 `continue-on-error: true` 到非关键步骤
- ✅ 使用 `if: always()` 确保产物总是上传
- ✅ 设置合理的超时时间

---

## 📊 当前状态

### CI 工作流

| 工作流 | 位置 | 状态 | Node |
|--------|------|------|------|
| frontend-ci.yml | .github/workflows/ | ✅ | 22 |
| ci.yml | itsm-frontend/.github/workflows/ | ✅ | 22 |
| backend-ci.yml | .github/workflows/ | ✅ | 22 |
| release.yml | .github/workflows/ | ✅ | 22 |
| frontend-tests.yml | .github/workflows/ | ✅ | 22 |
| automated-tests.yml | .github/workflows/ | ✅ | 22 |

### 最新提交

```
64f7ea1 - ci: unify frontend CI configuration (最新)
30c9aca - fix: remove npm cache config from frontend CI
```

---

## 🔗 相关链接

- **Root CI**: https://github.com/heidsoft/itsm/tree/main/.github/workflows
- **Frontend CI**: https://github.com/heidsoft/itsm/tree/main/itsm-frontend/.github/workflows
- **Actions**: https://github.com/heidsoft/itsm/actions

---

**维护者**: ITSM Team  
**最后更新**: 2026-02-28 14:10 CST
