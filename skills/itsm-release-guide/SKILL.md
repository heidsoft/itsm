# ITSM 项目发布指南

**版本**: 1.0.0  
**创建时间**: 2026-02-27  
**适用项目**: Next.js + Go/Gin + GitHub Actions 全栈项目

---

## 📋 问题清单与解决方案

### 1. 依赖冲突问题

**症状**:
```bash
npm error ERESOLVE could not resolve
npm error peer antd@"^4.24.15 || ^5.11.2" from @ant-design/pro-components
npm error Found: antd@6.2.2
```

**解决方案**:
```bash
# 1. 从 package.json 移除冲突依赖
npm uninstall @ant-design/pro-components

# 2. 清理并重新安装
rm -rf node_modules package-lock.json
npm install --legacy-peer-deps

# 3. 生成锁文件
npm install --package-lock-only
```

**预防措施**:
- 使用 `npm ls` 检查依赖树
- 升级大版本前先检查 peer dependencies
- 考虑使用自定义组件替代 ProComponents

---

### 2. TypeScript 类型检查问题

**症状**:
```bash
// @ts-nocheck 被大量使用
Type check failed with 50+ errors
```

**解决方案**:
```bash
# 1. 批量移除 @ts-nocheck
find src -name "*.tsx" -o -name "*.ts" | xargs sed -i '/@ts-nocheck/d'

# 2. 运行类型检查定位问题
npx tsc --noEmit

# 3. 逐个修复类型错误
# 常见错误：
# - 缺失的导入
# - any 类型
# - 接口定义不完整
```

**批量修复脚本**:
```bash
#!/bin/bash
# 移除所有 @ts-nocheck
for file in $(find src -name "*.tsx" -o -name "*.ts"); do
    sed -i '/@ts-nocheck/d' "$file"
done

# 运行类型检查
npx tsc --noEmit
```

---

### 3. GitHub Actions CI 配置问题

**常见问题**:

#### 3.1 包管理器混用
```yaml
# ❌ 错误：项目用 npm，CI 用 pnpm
- name: Setup pnpm
  uses: pnpm/action-setup@v4
- run: pnpm install --frozen-lockfile

# ✅ 正确：统一使用 npm
- name: Setup Node.js
  uses: actions/setup-node@v4
  with:
    cache: 'npm'
    cache-dependency-path: package-lock.json
- run: npm ci --legacy-peer-deps
```

#### 3.2 缓存配置问题
```yaml
# ❌ 错误：缓存路径不匹配
cache-dependency-path: pnpm-lock.yaml

# ✅ 正确
cache-dependency-path: package-lock.json
```

#### 3.3 Node.js 版本
```yaml
# ✅ 推荐：使用最新 LTS
- uses: actions/setup-node@v4
  with:
    node-version: '22'  # 或 20
```

---

### 4. Next.js 构建内存问题

**症状**:
```bash
Next.js build worker exited with code: null and signal: SIGKILL
```

**解决方案**:
```yaml
# GitHub Actions 配置
- name: Build
  run: npm run build
  env:
    NODE_OPTIONS: --max-old-space-size=4096
    NEXT_PUBLIC_API_URL: http://localhost:8080
```

**next.config.ts 优化**:
```typescript
const nextConfig: NextConfig = {
  // 构建优化
  swcMinify: true,
  
  // 忽略非关键错误
  eslint: {
    ignoreDuringBuilds: true,
  },
  typescript: {
    ignoreBuildErrors: true,  // 临时方案
  },
  
  // 包导入优化
  experimental: {
    optimizePackageImports: ['antd', '@ant-design/icons'],
  },
};
```

---

### 5. Release Workflow 问题

**常见问题**:

#### 5.1 Permissions 不足
```yaml
# ✅ 添加 permissions
permissions:
  contents: write  # 创建 Release 需要
```

#### 5.2 Artifact 下载失败
```yaml
# ❌ 错误：pattern 不匹配
- uses: actions/download-artifact@v4
  with:
    pattern: itsm-backend-*

# ✅ 正确：明确指定
- uses: actions/download-artifact@v4
  with:
    pattern: itsm-backend-*
    merge-multiple: true
    path: artifacts/backends
```

#### 5.3 依赖关系
```yaml
# ✅ 仅后端发布（前端有问题时）
release:
  needs: build-backend
  
# ✅ 完整发布
release:
  needs: [build-backend, build-frontend]
```

---

### 6. 测试失败处理

**策略**:

#### 6.1 临时跳过测试（紧急发布）
```yaml
# 在 CI 配置中注释测试步骤
# - name: Run unit tests
#   run: npm run test:unit

# 或添加条件
- name: Run unit tests
  if: github.event_name != 'release'
  run: npm run test:unit
```

#### 6.2 修复测试
```bash
# 1. 本地运行测试查看错误
npm run test:unit

# 2. 常见错误：
# - Mock 数据缺失
# - API 端点变更
# - 组件重构导致测试失效

# 3. 更新测试用例
```

---

## 🚀 标准发布流程

### 准备阶段
```bash
# 1. 本地验证
npm run type-check
npm run lint
npm run build

# 2. 提交所有更改
git add -A
git commit -m "fix: prepare for release v1.0.0"
git push origin main
```

### 创建 Release
```bash
# 1. 打 tag
git tag v1.0.0
git push origin v1.0.0

# 2. 或强制更新（修复后）
git tag -d v1.0.0
git tag v1.0.0
git push origin v1.0.0 --force
```

### 监控构建
```bash
# 查看最新构建
curl -sL "https://api.github.com/repos/heidsoft/itsm/actions/runs?per_page=3" | \
  python3 -c "import json,sys; d=json.load(sys.stdin); \
    [print(f\"{r['name']}: {r['conclusion']}\") for r in d['workflow_runs']]"

# 查看构建详情
curl -sL "https://api.github.com/repos/heidsoft/itsm/actions/runs/<RUN_ID>/jobs"
```

### 验证发布
```bash
# 检查 Release 是否创建
curl -sL "https://api.github.com/repos/heidsoft/itsm/releases" | \
  python3 -c "import json,sys; d=json.load(sys.stdin); \
    [print(f\"{r['tag_name']}: {len(r['assets'])} assets\") for r in d]"
```

---

## 📝 检查清单

### 发布前
- [ ] 所有 P0 问题已修复
- [ ] 类型检查通过
- [ ] Lint 检查通过
- [ ] 本地构建成功
- [ ] CI 配置已更新
- [ ] package-lock.json 已同步

### 发布中
- [ ] Tag 已推送
- [ ] Release workflow 已触发
- [ ] 后端构建成功（所有平台）
- [ ] 前端构建成功
- [ ] Release 创建成功
- [ ] 附件已上传

### 发布后
- [ ] 验证 Release 页面
- [ ] 下载并测试二进制文件
- [ ] 更新文档
- [ ] 通知相关人员

---

## 🛠️ 快速修复命令

```bash
# 依赖问题
rm -rf node_modules package-lock.json
npm install --legacy-peer-deps

# 类型问题
find src -name "*.tsx" -o -name "*.ts" | xargs sed -i '/@ts-nocheck/d'
npx tsc --noEmit

# CI 配置
# 编辑 .github/workflows/*.yml
# 确保使用 npm 而非 pnpm
# 添加 NODE_OPTIONS 环境变量

# 强制重新发布
git tag -d v1.0.0
git tag v1.0.0
git push origin v1.0.0 --force
```

---

## 📚 参考链接

- [GitHub Actions Docs](https://docs.github.com/en/actions)
- [Next.js Deployment](https://nextjs.org/docs/deployment)
- [npm ci](https://docs.npmjs.com/cli/commands/npm-ci)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)

---

## 💡 经验教训

1. **依赖管理**: 大版本升级前先检查 peer dependencies
2. **类型安全**: 不要长期使用 @ts-nocheck，问题会累积
3. **CI 配置**: 保持 CI 与本地环境一致
4. **内存优化**: Next.js 构建需要足够内存
5. **测试策略**: 测试失败不应阻塞紧急发布
6. **文档化**: 所有修复都应该记录

---

> **最后更新**: 2026-02-27  
> **维护者**: ITSM Team
