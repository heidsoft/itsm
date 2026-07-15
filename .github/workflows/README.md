# GitHub Actions 流水线

仓库只保留构建、测试、安全、文档和发布所需的核心流水线。普通 PR 按路径触发，避免无关模块重复消耗 CI 分钟。

## 保留的流水线

| Workflow | 触发时机 | 职责 |
|---|---|---|
| [`backend-ci.yml`](./backend-ci.yml) | 后端 push / PR | Go 格式、静态检查、构建、测试和依赖校验 |
| [`frontend-ci.yml`](./frontend-ci.yml) | 前端 push / PR | 单次安装后执行 ESLint、类型检查、Jest 和生产构建 |
| [`api-contract-check.yml`](./api-contract-check.yml) | API/路由变更 | 前后端路径与字段契约静态检查 |
| [`ga-gate.yml`](./ga-gate.yml) | 核心应用 push / PR | 启动核心 Compose 栈并执行健康检查和 API 烟测 |
| [`extensions-ci.yml`](./extensions-ci.yml) | Agent/CLI/Skill 变更 | 验证三个扩展模块 |
| [`docs.yml`](./docs.yml) | 文档变更 | 构建 MkDocs；main 分支部署 GitHub Pages |
| [`security.yml`](./security.yml) | push / PR / 每周 | gosec、Trivy、npm audit 和密钥扫描 |
| [`release.yml`](./release.yml) | `v*` tag | 构建 Release 产物并发布后端/前端 GHCR 镜像 |

## 设计边界

- `backend-ci` 与 `frontend-ci` 是代码质量事实来源，不在其他 PR 流水线重复执行相同测试。
- `ga-gate` 只负责集成验证，不重复单元测试和生产构建。
- 镜像与 GitHub Release 只从不可变的 `v*` tag 发布，不允许普通 push 发布 `latest`。
- Security 使用最小写权限；只有 SARIF、Pages 和 Release 任务获得对应写权限。
- Dependabot 继续提交依赖升级 PR，但不自动合并，由核心 CI 和维护者审核决定。

## 推荐的分支保护检查

在 `main` 分支保护中要求以下检查通过：

- `backend-ci / Lint`、`backend-ci / Build`、`backend-ci / Test`（后端变更）
- `frontend-ci / Frontend`（前端变更）
- `api-contract-check / API Contract Check`（API 契约变更）
- `ga-gate / GA Gate · Approved`（核心应用变更）

删除旧 workflow 后，应同步移除分支保护中已经不存在的 required check 名称。

## 本地等价验证

```bash
cd itsm-backend && go test ./... && go build ./main.go
cd ../itsm-frontend && npm run lint:check && npm run type-check
npm test -- --runInBand && npm run build

cd ..
docker compose -f docker-compose.dev.yml --profile dev config --quiet
```
