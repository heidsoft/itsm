# Husky Pre-commit Hooks

本项目使用 Husky + lint-staged 实现提交前自动代码检查。

## 配置说明

### 已安装的钩子

- **pre-commit**: 提交前自动运行代码检查和格式化

### 检查内容

1. **ESLint**: 检查代码质量问题并自动修复
2. **Prettier**: 统一代码格式
3. **TypeScript**: 类型检查 (通过 `npm run type-check`)

### 支持的文件类型

- `.ts`, `.tsx` - TypeScript/React 文件
- `.js`, `.jsx` - JavaScript/React 文件
- `.json`, `.md` - 配置文件和文档

## 使用方法

### 安装

```bash
npm install
```

`prepare` 脚本会自动设置 Husky 钩子。

### 手动测试

```bash
# 测试 pre-commit 钩子
npx lint-staged

# 运行所有检查
npm run lint:check
npm run type-check
```

### 跳过钩子 (紧急情况)

```bash
git commit --no-verify -m "紧急修复"
```

⚠️ **注意**: 仅在紧急情况下使用，后续需要手动修复代码质量问题。

## 故障排查

### 钩子未执行

确保 `.husky/` 目录有执行权限：
```bash
chmod +x .husky/pre-commit
```

### 检查失败

查看具体错误信息，修复后重新提交。常见问题：
- ESLint 报错：运行 `npm run lint` 自动修复
- TypeScript 报错：检查类型定义
- Prettier 格式化：运行 `npx prettier --write .`

## 配置自定义

修改 `package.json` 中的 `lint-staged` 配置来调整检查规则。

---

_配置时间：2026-03-02_
