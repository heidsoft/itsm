# ITSM Makefile
# 提供统一的构建、部署和管理命令

.PHONY: help build test run dev prod stop clean logs db-migrate db-seed lint fmt

# 默认目标
.DEFAULT_GOAL := help

# 变量定义
BACKEND_DIR := itsm-backend
FRONTEND_DIR := itsm-frontend
DOCKER_COMPOSE_DEV := docker-compose.dev.yml
DOCKER_COMPOSE_PROD := docker-compose.prod.yml

# 颜色定义
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

# ============================================
# 帮助信息
# ============================================
help: ## 显示此帮助信息
	@echo "$(GREEN)ITSM 项目管理命令$(NC)"
	@echo ""
	@echo "$(YELLOW)开发环境:$(NC)"
	@echo "  make dev-up         启动 Docker 开发环境"
	@echo "  make dev-down       停止开发环境"
	@echo "  make dev-logs       查看开发环境日志"
	@echo "  make dev-shell      进入后端容器"
	@echo ""
	@echo "$(YELLOW)构建:$(NC)"
	@echo "  make build          构建前后端镜像"
	@echo "  make build-backend  构建后端镜像"
	@echo "  make build-frontend 构建前端镜像"
	@echo ""
	@echo "$(YELLOW)本地运行:$(NC)"
	@echo "  make run            启动后端服务（本地）"
	@echo "  make frontend-run   启动前端服务（本地）"
	@echo ""
	@echo "$(YELLOW)测试:$(NC)"
	@echo "  make test           运行所有测试"
	@echo "  make test-backend   运行后端测试"
	@echo "  make test-frontend  运行前端测试"
	@echo "  make test-coverage  生成覆盖率报告"
	@echo ""
	@echo "$(YELLOW)代码质量:$(NC)"
	@echo "  make lint           代码检查"
	@echo "  make fmt            代码格式化"
	@echo ""
	@echo "$(YELLOW)数据库:$(NC)"
	@echo "  make db-migrate     运行数据库迁移"
	@echo "  make db-seed        导入初始数据"
	@echo "  make db-backup      备份数据库"
	@echo "  make db-restore     恢复数据库"
	@echo ""
	@echo "$(YELLOW)生产环境:$(NC)"
	@echo "  make prod-up        启动生产环境"
	@echo "  make prod-down      停止生产环境"
	@echo "  make prod-logs      查看生产环境日志"
	@echo ""
	@echo "$(YELLOW)维护:$(NC)"
	@echo "  make logs           查看所有日志"
	@echo "  make stop           停止所有服务"
	@echo "  make clean          清理构建产物和缓存"
	@echo "  make health         检查健康状态"
	@echo ""
	@echo "$(BLUE)示例:$(NC)"
	@echo "  make dev-up && make test  # 启动开发环境并运行测试"

# ============================================
# 开发环境 (Docker)
# ============================================
dev-up: ## 启动开发环境
	@echo "$(BLUE)🚀 启动开发环境...$(NC)"
	@cp .env.example .env 2>/dev/null || true
	docker compose -f $(DOCKER_COMPOSE_DEV) up -d --build
	@echo "$(GREEN)✅ 开发环境已启动$(NC)"
	@echo "   前端: http://localhost:3000"
	@echo "   后端: http://localhost:8090"
	@echo "   API 文档: http://localhost:8090/swagger"

dev-down: ## 停止开发环境
	@echo "$(YELLOW)⏹  停止开发环境...$(NC)"
	docker compose -f $(DOCKER_COMPOSE_DEV) down
	@echo "$(GREEN)✅ 开发环境已停止$(NC)"

dev-restart: dev-down dev-up ## 重启开发环境

dev-logs: ## 查看开发环境日志
	docker compose -f $(DOCKER_COMPOSE_DEV) logs -f

dev-logs-backend: ## 查看后端日志
	docker compose -f $(DOCKER_COMPOSE_DEV) logs -f itsm-backend

dev-logs-frontend: ## 查看前端日志
	docker compose -f $(DOCKER_COMPOSE_DEV) logs -f itsm-frontend

dev-shell: ## 进入后端容器
	docker exec -it itsm-backend sh

dev-db-shell: ## 进入数据库容器
	docker exec -it itsm-postgres psql -U itsm

# ============================================
# 构建
# ============================================
build: build-backend build-frontend ## 构建所有镜像

build-backend: ## 构建后端镜像
	@echo "$(BLUE)🔨 构建后端镜像...$(NC)"
	docker build -t itsm-backend:dev -f Dockerfile.backend ./$(BACKEND_DIR)
	@echo "$(GREEN)✅ 后端镜像构建完成$(NC)"

build-frontend: ## 构建前端镜像
	@echo "$(BLUE)🔨 构建前端镜像...$(NC)"
	docker build -t itsm-frontend:dev -f Dockerfile.frontend ./$(FRONTEND_DIR)
	@echo "$(GREEN)✅ 前端镜像构建完成$(NC)"

# ============================================
# 本地运行（不使用 Docker）
# ============================================
run: ## 启动后端服务（本地，需先有数据库）
	@echo "$(BLUE)🏃 启动后端服务...$(NC)"
	cd $(BACKEND_DIR) && go run main.go

run-dev: ## 启动后端（开发模式，使用 air 热重载）
	@if ! command -v air >/dev/null; then \
		echo "$(YELLOW)⚠️  air 未安装，正在安装...$(NC)"; \
		go install github.com/air-verse/air@latest; \
	fi
	cd $(BACKEND_DIR) && air

frontend-run: ## 启动前端（本地）
	@echo "$(BLUE)🏃 启动前端服务...$(NC)"
	cd $(FRONTEND_DIR) && npm run dev

frontend-build: ## 构建前端（生产）
	cd $(FRONTEND_DIR) && npm run build

# ============================================
# 测试
# ============================================
test: test-backend test-frontend ## 运行所有测试

test-backend: ## 运行后端测试
	@echo "$(BLUE)🧪 运行后端测试...$(NC)"
	cd $(BACKEND_DIR) && go test ./... -v -count=1
	@echo "$(GREEN)✅ 后端测试完成$(NC)"

test-backend-coverage: ## 后端测试覆盖率
	cd $(BACKEND_DIR) && go test ./... -coverprofile=coverage.out
	cd $(BACKEND_DIR) && go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)📊 覆盖率报告: $(BACKEND_DIR)/coverage.html$(NC)"

test-frontend: ## 运行前端测试
	@echo "$(BLUE)🧪 运行前端测试...$(NC)"
	cd $(FRONTEND_DIR) && npm test -- --watchAll=false
	@echo "$(GREEN)✅ 前端测试完成$(NC)"

test-frontend-coverage: ## 前端测试覆盖率
	cd $(FRONTEND_DIR) && npm run test:coverage
	@echo "$(GREEN)📊 覆盖率报告在 $(FRONTEND_DIR)/coverage/lcov-report/index.html$(NC)"

# ============================================
# 代码质量
# ============================================
lint: lint-backend lint-frontend ## 代码检查

lint-backend: ## 后端代码检查
	@echo "$(BLUE)🔍 检查后端代码...$(NC)"
	cd $(BACKEND_DIR) && go vet ./...
	cd $(BACKEND_DIR) && staticcheck ./...
	cd $(BACKEND_DIR) && gofmt -d . | grep -v '^$$' || echo "$(GREEN)✅ 代码格式正确$(NC)"

lint-frontend: ## 前端代码检查
	@echo "$(BLUE)🔍 检查前端代码...$(NC)"
	cd $(FRONTEND_DIR) && npm run lint
	cd $(FRONTEND_DIR) && npm run type-check

fmt: fmt-backend fmt-frontend ## 代码格式化

fmt-backend: ## 格式化后端代码
	cd $(BACKEND_DIR) && gofmt -w .
	cd $(BACKEND_DIR) && gofumpt -w .

fmt-frontend: ## 格式化前端代码
	cd $(FRONTEND_DIR) && npm run format

# ============================================
# 数据库
# ============================================
db-migrate: ## 运行数据库迁移
	@echo "$(BLUE)🗄️  运行数据库迁移...$(NC)"
	docker compose -f $(DOCKER_COMPOSE_DEV) exec -T itsm-backend go run cmd/migrate/main.go
	@echo "$(GREEN)✅ 迁移完成$(NC)"

db-migrate-down: ## 回滚迁移（谨慎）
	@echo "$(YELLOW)⚠️  回滚最后一次迁移...$(NC)"
	docker compose -f $(DOCKER_COMPOSE_DEV) exec -T itsm-backend go run cmd/migrate/main.go -down

db-seed: ## 导入初始数据
	@echo "$(BLUE)🌱 导入初始数据...$(NC)"
	docker compose -f $(DOCKER_COMPOSE_DEV) exec -T itsm-backend go run cmd/seed/main.go
	@echo "$(GREEN)✅ 初始数据导入完成$(NC)"

db-backup: ## 备份数据库
	@echo "$(BLUE)💾 备份数据库...$(NC)"
	@mkdir -p ./backups
	docker compose -f $(DOCKER_COMPOSE_DEV) exec -T itsm-postgres pg_dump -U itsm itsm > ./backups/itsm_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "$(GREEN)✅ 备份完成: ./backups/$(NC)"

db-restore: ## 恢复数据库（需提供备份文件路径）
	@read -p "请输入备份文件路径: " backup_file; \
	docker compose -f $(DOCKER_COMPOSE_DEV) exec -T itsm-postgres psql -U itsm itsm < $$backup_file

db-shell: ## 进入数据库 shell
	docker compose -f $(DOCKER_COMPOSE_DEV) exec -T itsm-postgres psql -U itsm itsm

# ============================================
# 生产环境
# ============================================
prod-up: ## 启动生产环境
	@echo "$(BLUE)🚀 启动生产环境...$(NC)"
	@if [ ! -f .env.prod ]; then \
		echo "$(RED)❌ 请先创建 .env.prod 文件$(NC)"; \
		exit 1; \
	fi
	docker compose -f $(DOCKER_COMPOSE_PROD) up -d --build
	@echo "$(GREEN)✅ 生产环境已启动$(NC)"
	@echo "   前端: http://localhost:3000"
	@echo "   后端: http://localhost:8090"

prod-down: ## 停止生产环境
	docker compose -f $(DOCKER_COMPOSE_PROD) down

prod-logs: ## 查看生产环境日志
	docker compose -f $(DOCKER_COMPOSE_PROD) logs -f

prod-restart: ## 重启生产环境
	docker compose -f $(DOCKER_COMPOSE_PROD) restart

# ============================================
# 维护
# ============================================
logs: ## 查看所有服务日志
	docker compose -f $(DOCKER_COMPOSE_DEV) logs --tail=100

stop: dev-down ## 停止所有服务（别名）

clean: ## 清理构建产物、缓存、Docker 资源
	@echo "$(YELLOW)🧹 清理缓存和构建产物...$(NC)"
	# 清理后端
	cd $(BACKEND_DIR) && rm -rf bin/ coverage.out coverage.html
	# 清理前端
	cd $(FRONTEND_DIR) && rm -rf .next/ coverage/ dist/ build/
	# 清理 Docker（谨慎）
	docker system prune -f --volumes 2>/dev/null || true
	@echo "$(GREEN)✅ 清理完成$(NC)"

health: ## 检查服务健康状态
	@echo "$(BLUE)🏥 检查健康状态...$(NC)"
	@echo ""
	@echo "后端健康:"
	@curl -s http://localhost:8090/health || echo "$(RED)❌ 后端不可用$(NC)"
	@echo ""
	@echo "数据库:"
	@docker exec itsm-postgres pg_isready -U itsm 2>/dev/null || echo "$(RED)❌ 数据库不可用$(NC)"
	@echo ""
	@echo "Redis:"
	@docker exec itsm-redis redis-cli ping 2>/dev/null || echo "$(RED)❌ Redis 不可用$(NC)"

# ============================================
# 工具
# ============================================
proto-gen: ## 生成 protobuf 代码（如使用）
	@echo "待实现"

api-gen: ## 生成 API 代码（如使用 openapi-generator）
	@echo "待实现"

swagger: ## 生成 Swagger 文档
	@echo "$(BLUE)📄 生成 Swagger 文档...$(NC)"
	cd $(BACKEND_DIR) && swag init -g cmd/main.go -o docs --parseDependency --parseInternal
	@echo "$(GREEN)✅ Swagger 文档已生成: $(BACKEND_DIR)/docs/swagger.json$(NC)"

install-tools: ## 安装开发工具（Go、Node 依赖）
	@echo "$(BLUE)🔧 安装开发工具...$(NC)"
	# 后端工具
	cd $(BACKEND_DIR) && go mod download
	cd $(BACKEND_DIR) && go install github.com/swaggo/swag/cmd/swag@latest
	cd $(BACKEND_DIR) && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	cd $(BACKEND_DIR) && go install github.com/go-delve/delve/cmd/dlve@latest
	cd $(BACKEND_DIR) && go install github.com/air-verse/air@latest
	cd $(BACKEND_DIR) && go install mvdan.cc/gofumpt@latest
	cd $(BACKEND_DIR) && go install honnef.co/go/tools/cmd/staticcheck@latest
	# 前端工具
	cd $(FRONTEND_DIR) && npm install
	@echo "$(GREEN)✅ 工具安装完成$(NC)"

# ============================================
# CI/CD（本地模拟）
# ============================================
ci: lint test ## 模拟 CI 流程（lint + test）
	@echo "$(GREEN)✅ CI 流程通过$(NC)"

docker-push: ## 推送镜像到仓库（需先登录）
	@echo "$(BLUE)📤 推送镜像...$(NC)"
	docker tag itsm-backend:dev registry.example.com/itsm-backend:latest
	docker push registry.example.com/itsm-backend:latest
	docker tag itsm-frontend:dev registry.example.com/itsm-frontend:latest
	docker push registry.example.com/itsm-frontend:latest
	@echo "$(GREEN)✅ 镜像推送完成$(NC)"

# 快速开始（第一次运行）
quickstart: dev-up ## 快速启动（开发环境）
	@echo ""
	@echo "$(GREEN)🎉 环境已就绪！$(NC)"
	@echo ""
	@echo "下一步:"
	@echo "  1. 访问 http://localhost:3000"
	@echo "  2. 使用 admin/admin123 登录"
	@echo "  3. 查看文档: http://localhost:8090/swagger"
	@echo ""
	@echo "常用命令:"
	@echo "  make logs       查看日志"
	@echo "  make db-shell   进入数据库"
	@echo "  make stop       停止服务"
	@echo ""
