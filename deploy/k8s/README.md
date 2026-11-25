# Kubernetes 部署清单（起步脚手架）

本目录提供 Docker Compose 之外的 Kubernetes 部署选项。清单为**起步脚手架**，生产落地前需按集群环境校验并调整：

- 镜像标签 `ghcr.io/heidsoft/itsm-backend:1.6.60` / `itsm-frontend:1.6.60` —— 改为你的固定标签（不要用 `latest`）；本目录仍是起步脚手架，不是 HA 方案。
- `02-secrets.yaml` 为占位符，**部署前必须替换**（推荐 sealed-secrets / external-secrets / vault）。
- PVC 的 `storageClassName` 依赖集群默认；如需指定请显式设置。
- 若后端镜像依赖 `config.yaml`，请将 `itsm-backend/config.yaml.example` 挂载为 ConfigMap 并挂到容器 `/app/config`（或对应路径）。
- Ingress 示例用 `nginx` ingressClass，域名 `itsm.example.com` 改为真实域名，并自行配置 TLS（cert-manager 或手动 Secret）。
- `22-migrate-job.yaml` 的迁移命令需与 `itsm-init` 入口一致；建议先用该 Job 完成迁移，再上线 backend/worker。

## 部署顺序

```bash
kubectl apply -f 00-namespace.yaml
kubectl apply -f 01-configmap.yaml
# 1) 先创建真实 secrets（见 02-secrets.yaml 头部命令），或 kubectl apply -f 02-secrets.yaml 后立刻改值
kubectl apply -f 02-secrets.yaml
kubectl apply -f 10-postgres.yaml 11-redis.yaml
# 2) 迁移数据库
kubectl apply -f 22-migrate-job.yaml
kubectl wait --for=condition=complete job/itsm-migrate -n itsm --timeout=300s
# 3) 应用层
kubectl apply -f 20-backend.yaml 21-worker.yaml 30-frontend.yaml 40-ingress.yaml
```

## 升级

1. 修改镜像标签（backend/frontend）为新版本。
2. 重新应用 `22-migrate-job.yaml`（或新建带版本后缀的 Job）完成迁移。
3. 滚动更新：`kubectl -n itsm rollout restart deployment/itsm-backend deployment/itsm-worker deployment/itsm-frontend`。
4. 破坏性变更与回滚见仓库根目录 [UPGRADE.md](../../UPGRADE.md)。

## 未覆盖（按需自补）

- **itsm-ai-service（受限分诊 sidecar）**：未部署时后端/worker 的 AI 分诊走降级路径。部署后确认 `01-configmap.yaml` 的 `GUIDANCE_URL` 指向真实 Service，并在 `02-secrets.yaml` 配置与 sidecar 同值的 `AI_SERVICE_TOKEN`（sidecar fail-closed，令牌不一致则分诊 401）。
- MinIO / 对象存储（默认部署不使用，对应 Compose `storage` profile）。
- Grafana / Prometheus monitoring 堆栈。
- HPA、PodDisruptionBudget、资源配额。
- 多副本 Postgres / Redis（生产建议托管或集群版）。
