# ITSM Release Guide Skill

这个 skill 提供了 ITSM 项目（Next.js + Go/Gin）的完整发布指南和问题解决方案。

## 使用场景

- 🚀 首次发布 ITSM 项目
- 🔧 修复 CI/CD 构建问题
- 📦 创建 GitHub Release
- 🐛 解决依赖冲突
- ✅ 通过类型检查

## 快速开始

```bash
# 1. 查看完整指南
cat skills/itsm-release-guide/SKILL.md

# 2. 检查当前状态
cd itsm-frontend
npm run type-check
npm run lint
npm run build

# 3. 修复常见问题
./skills/itsm-release-guide/fix-common-issues.sh
```

## 核心内容

### 1. 依赖冲突解决
- 识别 peer dependencies 冲突
- 使用 --legacy-peer-deps
- 清理和重新安装

### 2. TypeScript 类型修复
- 批量移除 @ts-nocheck
- 定位和修复类型错误
- 配置构建时忽略策略

### 3. CI/CD 配置优化
- 统一包管理器（npm vs pnpm）
- 正确的缓存配置
- Node.js 版本选择

### 4. 构建优化
- 内存配置（NODE_OPTIONS）
- Next.js 配置优化
- 减少构建时间

### 5. Release 流程
- 创建和推送 tag
- 配置 GitHub Actions
- 验证发布结果

## 相关脚本

- `fix-common-issues.sh` - 批量修复常见问题
- `check-build-status.sh` - 检查 GitHub Actions 状态
- `create-release.sh` - 自动化发布流程

## 维护

当遇到新问题时：
1. 记录问题和解决方案
2. 更新 SKILL.md
3. 添加快速修复命令
4. 更新检查清单

---

**版本**: 1.0.0  
**创建**: 2026-02-27
