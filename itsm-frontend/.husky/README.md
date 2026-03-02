# Husky Git Hooks

本项目使用 Husky + lint-staged 实现 Git 钩子自动化，确保代码质量和提交规范。

## 配置说明

### 已安装的钩子

| 钩子 | 说明 | 触发时机 |
|------|------|----------|
| **pre-commit** | 代码检查和格式化 | 执行 `git commit` 时 |
| **commit-msg** | 提交信息格式验证 | 提交信息输入完成后 |

### pre-commit 检查内容

1. **ESLint**: 检查代码质量问题并自动修复
2. **Prettier**: 统一代码格式
3. **lint-staged**: 仅检查暂存文件

### commit-msg 格式规范

提交信息必须遵循以下格式：
```
<type>(<scope>): <subject>
```

**type 可选值:**
- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式 (不影响代码运行)
- `refactor`: 代码重构 (非新功能或 bug 修复)
- `test`: 添加或修改测试
- `chore`: 构建过程或辅助工具变动
- `perf`: 性能优化
- `ci`: CI 配置
- `build`: 构建系统或外部依赖
- `revert`: 回滚提交

**示例:**
```bash
feat(auth): 添加用户登录功能
fix(api): 修复工单创建接口错误
docs: 更新 API 文档
refactor(frontend): 重构事件管理组件
```

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

# 测试 commit-msg 钩子
echo "feat(test): 测试提交信息" | npx commitlint --stdin

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
chmod +x .husky/commit-msg
```

### pre-commit 检查失败

查看具体错误信息，修复后重新提交。常见问题：
- ESLint 报错：运行 `npm run lint` 自动修复
- Prettier 格式化：运行 `npx prettier --write .`

### commit-msg 格式错误

按照提示的格式修改提交信息。确保：
- 包含 type (如 feat, fix, docs)
- 可选 scope 用括号包裹
- 冒号后跟一个空格
- 描述不超过 72 字符

## 配置自定义

### 修改 lint-staged 规则

编辑 `package.json` 中的 `lint-staged` 配置：

```json
{
  "lint-staged": {
    "*.{ts,tsx}": ["eslint --fix", "prettier --write"],
    "*.{js,jsx}": ["eslint --fix", "prettier --write"],
    "*.{json,md}": ["prettier --write"]
  }
}
```

### 添加新的 Git 钩子

```bash
# 在 .husky 目录下创建新钩子
npx husky add .husky/pre-push "npm test"
```

---

_最后更新：2026-03-03_
