# 云信号生产能力验证记录（2026-11-01）

> Status: verified
> 部署批次：`ac865c9ab` feat(cmdb): 云信号接入——安全漏洞与 CloudMonitor 告警进入 ITSM 主流程
> 验证环境：本机生产栈（`docker-compose.prod.yml`，镜像 `itsm-backend:v1.6.40` / `itsm-frontend:v1.6.40`，nginx :80）

## 结论

生产构建中云监控告警、安全漏洞、同步健康三类读面与前端页面均已上线，契约与空状态语义正确；
真实云数据链路（凭据解析、告警/漏洞入库、P1/P2 自动建单）因生产未配置云账号，本次未覆盖。

## 后端契约（只读）

| 用例 | 期望 | 实测 |
|:---|:---|:---|
| 未登录 `GET /api/v1/cloud-signals/sync-status` | 401 / 2001 | 401 / 2001 |
| 未登录 `GET /api/v1/cloud-signals/alerts` | 401 / 2001 | 401 / 2001 |
| 未登录 `GET /api/v1/cloud-signals/vulnerabilities` | 401 / 2001 | 401 / 2001 |
| 未登录 `GET /api/v1/cloud-signals/vulnerabilities/summary` | 401 / 2001 | 401 / 2001 |
| 登录后 `GET /sync-status`、`/alerts`、`/vulnerabilities` | 200，`{items,total,page,pageSize,totalPages}` | 200，五键齐全，`total=0` |
| 登录后 `GET /vulnerabilities/summary` | 200，`{bySeverity,byStatus,total}` | 200，结构一致 |
| `GET /alerts?status=bogus` | 400 / 1001 | 400 / 1001「告警过滤条件不合法」 |
| `GET /alerts?search=<201 字符>` | 400 / 1001（长度上限 200） | 400 / 1001 |
| `GET /vulnerabilities?severity=critical&severity=high` | 200（重复键多选） | 200 |

说明：`401` 而非 `404` 证明生产镜像已注册云信号路由；`/api/v1/cmdb/cloud-accounts` 返回 0 条，
因此 `sync-status` 为空与「没有账号可同步」一致，不构成同步失败误报。

## 前端页面（Playwright，真实表单登录）

- `/cmdb/cloud-signals` 渲染正常，页签为「云监控告警」「安全漏洞」
- 空状态显示「还没有云信号同步记录」及 `secret://` / `env://` 凭据配置指引，未伪装成同步成功
- 漏洞页签汇总卡（未修复合计 / 严重 / 高危 / 中危 / 低危 / 未知）与默认筛选「待修复、修复中」正确
- 实测请求 URL 为 `.../vulnerabilities?status=open&status=fixing&page=1&pageSize=20`，
  证明 `http-client` 的数组参数按重复键序列化在生产生效
- `/cmdb/cloud-resources` 正常渲染，新增「安全与告警」入口可见；无 4xx/5xx，
  控制台仅有导航时的 `auth/session ERR_ABORTED`（请求被页面切换中断，非服务端错误）

## 证据

- 告警页签空状态：`.ant-alert` 文本为「还没有云信号同步记录」，附 `secret://` / `env://` 配置指引
- 漏洞页签：请求 `GET /api/v1/cloud-signals/vulnerabilities?status=open&status=fixing&page=1&pageSize=20` 返回 200，
  汇总卡显示未修复合计与严重度分档，控制台无错误
- 页面截图已在本机留存，未入库（`.gitignore` 规则 `*.png` 明确排除图片）

## 未覆盖与限制

1. **真实云数据链路**：生产未配置云账号与凭据引用，告警/漏洞入库、P1/P2 自动建单、
   ECS 详情与漏洞关联均无真实数据可验；需要先在生产配置 `secret://`/`env://` 凭据。
2. **写操作**：认领/关闭告警、忽略/重开漏洞未在生产调用，避免污染生产数据；
   状态机与非法迁移语义由 `handlers/cloudsignal/service_test.go` 覆盖。
3. **观察项（非阻断）**：页面首屏在 `sync-status` 返回前会短暂落到「云信号同步正常」分支，
   数据到达后正确切换为空状态提示；如需消除闪烁，可改为加载态占位。
