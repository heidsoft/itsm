# GitHub Actions 流水线

仓库只保留构建、测试、安全、文档和发布所需的核心流水线。普通 PR 按路径触发，避免无关模块重复消耗 CI 分钟。

## 保留的流水线

| Workflow | 触发时机 | 职责 |
|---|---|---|
| [`backend-ci.yml`](./backend-ci.yml) | 后端 push / PR | Go 格式、静态检查、构建、测试和依赖校验 |
| [`frontend-ci.yml`](./frontend-ci.yml) | 前端 push / PR | 单次安装后执行 ESLint、类型检查、Jest 和生产构建 |
| [`api-contract-check.yml`](./api-contract-check.yml) | API/路由变更 | 前后端路径与字段契约静态检查 |
| [`test-coverage-guard.yml`](./test-coverage-guard.yml) | 源代码变更 | 强制「源码 X 改动 ⇒ 测试 X 必动」规则 |
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
- `test-coverage-guard / Source/Test Coverage Guard`（源代码变更）
- `ga-gate / GA Gate · Approved`（核心应用变更）

删除旧 workflow 后，应同步移除分支保护中已经不存在的 required check 名称。

## 本地等价验证

```bash
cd itsm-backend && go test ./... && go build ./main.go
cd ../itsm-frontend && npm run lint:check && npm run type-check
npm test -- --runInBand && npm run build

cd ..
node scripts/test-coverage-guard.js --base HEAD~1 --head HEAD
make check-contracts
```

## 前端 API 构建约定

- CI、Release 和标准 Nginx 部署保持 `NEXT_PUBLIC_API_URL` 为空。
- API client 路径已包含 `/api/v1/*`，浏览器通过同源 Nginx 转发。
- Next.js 服务端访问后端使用 `ITSM_BACKEND_URL`，不将容器主机名或开发机地址嵌入浏览器产物。
- 仅在不经反向代理的本机开发中，才设置 `NEXT_PUBLIC_API_URL=http://localhost:8090`。

## 可执行工程约束

`scripts/check-engineering-contracts.js` 是跨文件约定的单一检查入口，当前覆盖：

- 浏览器 API 默认同源，CI 和 Release 不得嵌入开发机后端地址。
- `.env.prod.example` 与 Next.js server-side 后端地址保持分层。
- 前端 Docker 构建不得携带 `.env.local` 等本机环境文件。
- Compose 使用后端真实读取的 CORS 变量和 IPv4 健康检查。
- MkDocs 依赖必须通过 `requirements-docs.txt` 可复现安装。

本地运行 `make check-contracts`；PR 中由 `api-contract-check / Engineering Contract Guard` 自动执行。新增跨文件规则时，必须同时增加 guard 的失败测试。
