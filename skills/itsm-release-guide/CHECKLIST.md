# ITSM 发布检查清单

## 📋 发布前检查

### 代码质量
- [ ] `npm run lint` 通过
- [ ] `npm run type-check` 通过
- [ ] `npm run test:unit` 通过（可选）
- [ ] `npm run build` 本地成功

### 依赖管理
- [ ] package.json 无冲突依赖
- [ ] package-lock.json 已同步
- [ ] node_modules 已清理并重新安装

### CI/CD 配置
- [ ] .github/workflows/frontend-ci.yml 使用 npm
- [ ] .github/workflows/release.yml 配置正确
- [ ] NODE_OPTIONS 已设置（--max-old-space-size=4096）
- [ ] permissions: contents: write 已添加

### Git 状态
- [ ] 所有更改已提交
- [ ] 代码已推送到 main 分支
- [ ] 无未解决的合并冲突

---

## 🚀 发布流程

### 1. 创建 Tag
```bash
git tag v1.0.0
git push origin v1.0.0
```

### 2. 监控构建
- [ ] Release workflow 已触发
- [ ] build-backend 成功（所有平台）
- [ ] build-frontend 成功
- [ ] release 任务成功
- [ ] deploy 任务成功

### 3. 验证发布
- [ ] Release 页面已创建
- [ ] 附件已上传（.zip 文件）
- [ ] Release notes 已生成

---

## ⚠️ 问题排查

### 构建失败
1. 查看 GitHub Actions 日志
2. 定位失败的步骤
3. 参考 SKILL.md 中的解决方案
4. 修复后重新推送

### 依赖冲突
```bash
cd itsm-frontend
rm -rf node_modules package-lock.json
npm install --legacy-peer-deps
```

### 类型错误
```bash
# 批量移除 @ts-nocheck
find src -name "*.tsx" -o -name "*.ts" | xargs sed -i '/@ts-nocheck/d'

# 运行类型检查
npx tsc --noEmit
```

### Release 失败
1. 检查 permissions 配置
2. 验证 artifact 下载路径
3. 确认 tag 已推送
4. 重新触发 workflow

---

## 📊 构建状态监控

```bash
# 查看最新构建
curl -sL "https://api.github.com/repos/heidsoft/itsm/actions/runs?per_page=3" | \
  python3 -c "import json,sys; d=json.load(sys.stdin); \
    [print(f\"{r['name']}: {r['conclusion']}\") for r in d['workflow_runs']]"

# 查看 Release
curl -sL "https://api.github.com/repos/heidsoft/itsm/releases" | \
  python3 -c "import json,sys; d=json.load(sys.stdin); \
    [print(f\"{r['tag_name']}: {len(r['assets'])} assets\") for r in d]"
```

---

## ✅ 发布完成确认

- [ ] Release v1.0.0 已创建
- [ ] 后端二进制文件已上传（6 个平台）
- [ ] 前端构建已上传
- [ ] Release URL 已分享给团队
- [ ] 部署文档已更新

---

**最后更新**: 2026-02-27  
**版本**: v1.0.0
