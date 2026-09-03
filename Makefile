# ITSM Makefile - 构建和部署自动化
.PHONY: help build build-backend build-frontend deploy test clean

# 默认版本
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "latest")
ENV_FILE ?= .env.prod
COMPOSE_FILE ?= docker-compose.prod.yml

# 颜色
BLUE = \033[0;34m
GREEN = \033[0;32m
NC = \033[0m

help: ## 显示帮助信息
	@echo "$(BLUE)ITSM 构建系统$(NC)"
	@echo ""
	@echo "用法: make [target]"
	@echo ""
	@echo "目标:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: build-backend build-frontend ## 构建所有镜像
	@echo "$(GREEN)构建完成！$(NC)"

build-backend: ## 构建后端镜像
	@echo "$(BLUE)构建后端镜像...$(NC)"
	DOCKER_BUILDKIT=1 docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) \
		build --build-arg "VERSION=$(VERSION)" itsm-backend

build-frontend: ## 构建前端镜像
	@echo "$(BLUE)构建前端镜像...$(NC)"
	DOCKER_BUILDKIT=1 docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) \
		build --build-arg "VERSION=$(VERSION)" itsm-frontend

build-parallel: ## 并行构建前后端
	@echo "$(BLUE)并行构建前后端...$(NC)"
	DOCKER_BUILDKIT=1 COMPOSE_DOCKER_CLI_BUILD=1 \
		docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) \
		build --parallel --build-arg "VERSION=$(VERSION)"

build-no-cache: ## 不使用缓存构建
	@echo "$(BLUE)无缓存构建...$(NC)"
	DOCKER_BUILDKIT=1 docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) \
		build --no-cache --build-arg "VERSION=$(VERSION)"

deploy: ## 部署到生产环境
	@echo "$(BLUE)部署到生产环境...$(NC)"
	docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) down
	docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) up -d
	@echo "$(GREEN)部署完成！$(NC)"

deploy-backend: ## 只部署后端
	@echo "$(BLUE)部署后端...$(NC)"
	docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) up -d itsm-backend itsm-worker

deploy-frontend: ## 只部署前端
	@echo "$(BLUE)部署前端...$(NC)"
	docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) up -d itsm-frontend

restart: ## 重启所有服务
	@echo "$(BLUE)重启服务...$(NC)"
	docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) restart

restart-backend: ## 重启后端
	docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) restart itsm-backend itsm-worker

restart-frontend: ## 重启前端
	docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) restart itsm-frontend

logs: ## 查看所有日志
	docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) logs -f

logs-backend: ## 查看后端日志
	docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) logs -f itsm-backend

logs-frontend: ## 查看前端日志
	docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) logs -f itsm-frontend

test: test-backend test-frontend ## 运行所有测试
	@echo "$(GREEN)测试完成！$(NC)"

test-backend: ## 运行后端测试
	@echo "$(BLUE)运行后端测试...$(NC)"
	cd itsm-backend && go test ./...

test-frontend: ## 运行前端测试
	@echo "$(BLUE)运行前端测试...$(NC)"
	cd itsm-frontend && npm test

test-unit: ## 运行单元测试
	cd itsm-backend && go test ./...
	cd itsm-frontend && npm run test:unit

test-e2e: ## 运行E2E测试
	cd itsm-frontend && npm run test:e2e

lint: lint-backend lint-frontend ## 运行所有lint
	@echo "$(GREEN)Lint完成！$(NC)"

lint-backend: ## 运行后端lint
	@echo "$(BLUE)运行后端lint...$(NC)"
	cd itsm-backend && golangci-lint run

lint-frontend: ## 运行前端lint
	@echo "$(BLUE)运行前端lint...$(NC)"
	cd itsm-frontend && npm run lint

type-check: ## TypeScript类型检查
	cd itsm-frontend && npm run type-check

health: ## 健康检查
	@echo "$(BLUE)检查服务状态...$(NC)"
	@curl -s http://localhost/api/v1/health | jq . || echo "后端不健康"
	@curl -s http://localhost | head -1 || echo "前端不健康"

# ========================
# 开发环境优化命令
# ========================

dev-start-docker: ## 启动优化后的Docker开发环境（启用BuildKit + 前端持久化缓存）
	@echo "$(BLUE)启动Docker开发环境（优化模式）...$(NC)"
	@DOCKER_BUILDKIT=1 docker compose -f docker-compose.dev.yml up -d
	@echo "$(GREEN)优化：BuildKit缓存 + .next持久化已启用$(NC)"
	@echo "$(GREEN)提示：前端代码变更会自动热重载，但建议重启时执行 dev-rebuild 清理缓存$(NC)"

dev-rebuild: ## 重建开发环境镜像（启用BuildKit缓存）
	@echo "$(BLUE)重建开发环境（保留缓存）...$(NC)"
	@DOCKER_BUILDKIT=1 docker compose -f docker-compose.dev.yml build --pull itsm-backend itsm-frontend
	@docker compose -f docker-compose.dev.yml up -d itsm-backend itsm-frontend
	@echo "$(GREEN)重建完成$(NC)"

dev-backend-local: ## 本地直接运行后端（编译产物），仅容器提供DB/Redis/Minio
	@echo "$(BLUE)启动本地后端模式（DB/Redis/Minio走容器）...$(NC)"
	@docker compose -f docker-compose.dev.yml up -d postgres redis minio itsm-init
	@echo "$(BLUE)等待DB就绪...$(NC)"
	@sleep 3
	@echo "$(BLUE)请在另一个终端运行：cd itsm-backend && go run ./cmd/server$(NC)"
	@echo "$(GREEN)后端监听 localhost:8090$(NC)"
	@echo "$(GREEN)前端：cd itsm-frontend && npm run dev$(NC)"

dev-stop: ## 停止开发环境
	@echo "$(BLUE)停止开发环境...$(NC)"
	docker compose -f docker-compose.dev.yml down
	@echo "$(GREEN)停止完成$(NC)"

dev-logs: ## 查看开发环境日志
	docker compose -f docker-compose.dev.yml logs -f

dev-status: ## 显示开发环境状态
	docker compose -f docker-compose.dev.yml ps

dev-reset: ## 重置开发环境（清除所有数据）
	@echo "$(RED)警告：将删除所有开发数据！$(NC)"
	@read -p "确认? (y/N) " -n 1 -r; echo; if [[ ! $$REPLY =~ ^[Yy]$$ ]]; then exit 1; fi
	docker compose -f docker-compose.dev.yml down -v
	rm -rf itsm-frontend/.next
	@echo "$(GREEN)重置完成，执行 make dev-start-docker 重新启动$(NC)"

dev-frontend-only: ## 仅启动前端开发服务器（需要后端已运行）
	@echo "$(BLUE)启动前端热重载开发服务器...$(NC)"
	cd itsm-frontend && npm run dev

status: ## 显示服务状态
	docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) ps

clean: ## 清理构建缓存
	@echo "$(BLUE)清理构建缓存...$(NC)"
	docker system prune -f
	docker volume prune -f
	rm -rf .build-cache
	@echo "$(GREEN)清理完成！$(NC)"

clean-all: ## 清理所有（包括数据）
	@echo "$(BLUE)清理所有数据...$(NC)"
	docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) down -v
	docker system prune -af
	@echo "$(GREEN)清理完成！$(NC)"

db-shell: ## 连接数据库
	docker exec -it itsm-postgres-prod psql -U itsm -d itsm_prod

redis-shell: ## 连接Redis
	docker exec -it itsm-redis-prod redis-cli -a $(REDIS_PASSWORD)

backend-shell: ## 进入后端容器
	docker exec -it itsm-backend-prod sh

frontend-shell: ## 进入前端容器
	docker exec -it itsm-frontend-prod sh

backup: ## 备份数据库
	@echo "$(BLUE)备份数据库...$(NC)"
	docker exec itsm-postgres-prod pg_dump -U itsm itsm_prod > backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "$(GREEN)备份完成！$(NC)"

restore: ## 恢复数据库（需要指定BACKUP_FILE）
	@if [ -z "$(BACKUP_FILE)" ]; then echo "用法: make restore BACKUP_FILE=backup.sql"; exit 1; fi
	@echo "$(BLUE)恢复数据库...$(NC)"
	docker exec -i itsm-postgres-prod psql -U itsm itsm_prod < $(BACKUP_FILE)
	@echo "$(GREEN)恢复完成！$(NC)"

version: ## 显示版本信息
	@echo "版本: $(VERSION)"
	@echo "环境文件: $(ENV_FILE)"
	@echo "Compose文件: $(COMPOSE_FILE)"
