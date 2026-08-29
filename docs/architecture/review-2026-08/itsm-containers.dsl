workspace "ITSM Platform" "当前容器视图 — 架构评审 2026-08-29（current-state）" {

    model {
        operator = person "运营/终端用户" "工单、事件、服务请求、知识库等日常操作"
        tenantAdmin = person "租户管理员" "租户配置、SLA 策略、RBAC、服务目录"
        mspOperator = person "MSP 运营商" "跨客户租户管理（委托范围）"

        itsm = softwareSystem "ITSM Platform" "AI-Native IT 服务管理平台（模块化单体，ADR-001）" {
            nginx = container "Nginx" "反向代理 / TLS 终结 / 端口 80、443" "Nginx"
            frontend = container "ITSM Frontend" "管理控制台（App Router，40 个业务路由）" "Next.js / React / Ant Design / Tailwind"
            backend = container "ITSM Backend API" "领域规则、RBAC、租户隔离、BPMN、LLM Gateway（端口 8090）" "Go / Gin / Ent"
            worker = container "ITSM Worker" "SLA 计时、升级、命令 outbox 消费（同镜像，ITSM_PROCESS_MODE=worker）" "Go"
            postgres = container "PostgreSQL 17 + pgvector" "业务数据 + RLS + RAG 向量 + operational_commands outbox" "PostgreSQL"
            redis = container "Redis 7" "缓存、限流、令牌撤销、分布式锁" "Redis"
            minio = container "MinIO（可选）" "对象存储，profile=storage，默认不启动" "MinIO"

            nginx -> frontend "路由 SSR / 静态资源" "HTTP"
            nginx -> backend "代理 /api" "HTTP"
            frontend -> backend "REST /api/v1（78 个 API client）" "HTTP/JSON"
            backend -> postgres "读写：应用层 tenant predicate + RLS 纵深防御" "SQL"
            backend -> redis "缓存 / 限流 / 会话" "RESP"
            backend -> minio "附件（可选路径）" "S3 API"
            worker -> postgres "消费 outbox、SLA 扫描、调度" "SQL"
            worker -> redis "lease / 分布式锁" "RESP"
        }

        operator -> nginx "使用平台" "HTTPS"
        tenantAdmin -> nginx "管理配置" "HTTPS"
        mspOperator -> nginx "跨租户运营" "HTTPS"

        ollama = container "Ollama（仅开发）" "本地 LLM 推理，dev compose profile=ai" "Ollama" {
            tags "Assumed"
        }
        backend -> ollama "LLM 调用（开发环境）" "HTTP"

        aiService = container "itsm-ai-service" "Python FastAPI：triage / RCA / 风险预测" "FastAPI" {
            tags "Unknown"
        }
        agentSvc = container "itsm-agent" "Agent 运行时（Go，含预编译二进制）" "Go" {
            tags "Unknown"
        }
        ragSvc = container "itsm-rag" "Python RAG 模块（独立 compose）" "Python" {
            tags "Unknown"
        }
        backend -> aiService "未被部署编排，调用关系未证实" "unknown"
        backend -> agentSvc "未被部署编排，调用关系未证实" "unknown"
    }

    views {
        container itsm "Containers-2026-08" "当前容器拓扑（浮动资产标记 Unknown）" {
            include *
            autoLayout
        }
        styles {
            element "Unknown" {
                color #b45309
                stroke #b45309
            }
            element "Assumed" {
                stroke #9ca3af
            }
        }
        theme default
    }
}
