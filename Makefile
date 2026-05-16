# ITSM Makefile
# 提供统一的构建、部署和管理命令

.PHONY: help build test run dev prod stop clean logs db-migrate db-seed lint fmt
.PHONY: dev-up dev-down dev-logs dev-shell dev-db-shell dev-restart
.PHONY: prod-up prod-down prod-logs prod-status prod-health prod-backup prod-rollback prod-dry-run prod-restart

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
NC := \033[0m

# ============================================
# 帮助信息
# ============================================
help: ## 显示此帮助信息
	@echo "$(GREEN)ITSM 项目管理命令$(NC)"
	@echo ""
	@echo "$(YELLOW)快速开始:$(NC)"
	@echo "  make quickstart     一键启动开发环境（推荐新用户）"
	@echo ""
	@echo "$(YELLOW)开发环境:$(NC)"
	@echo "  make dev-up         启动开发环境（自动检测 Docker/本地）"
	@echo "  make dev-down       停止开发环境"
	@echo "  make dev-restart    重启开发环境"
	@echo "  make dev-logs       查看开发环境日志"
	@echo "  make dev-shell      进入后端容器"
	@echo "  make dev-db-shell   进入数据库容器"
	@echo "  make dev-doctor     诊断开发环境问题"
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
	@echo ""
	@echo "$(YELLOW)代码质量:$(NC)"
	@echo "  make lint           代码检查"
	@echo "  make fmt            代码格式化"
	@echo ""
	@echo "$(YELLOW)数据库:$(NC)"
	@echo "  make db-migrate     运行数据库迁移"
	@echo "  make db-seed        导入初始数据"
	@echo "  make db-backup      备份数据库"
	@echo ""
	@echo "$(YELLOW)生产环境:$(NC)"
	@echo "  make prod-up        部署生产环境（含验证）"
	@echo "  make prod-down      停止生产环境"
	@echo "  make prod-logs      查看生产环境日志"
	@echo "  make prod-status    查看生产环境状态"
	@echo "  make prod-health    生产环境健康检查"
	@echo "  make prod-backup    备份生产数据库"
	@echo "  make prod-rollback  回滚到上一个版本"
	@echo "  make prod-dry-run   预览生产部署"
	@echo ""
	@echo "$(YELLOW)维护:$(NC)"
	@echo "  make clean          清理构建产物和缓存"
	@echo "  make health         检查健康状态"

# ============================================
# 开发环境
# ============================================
dev-up: ## 启动开发环境
	@./scripts/deploy-dev.sh up

dev-down: ## 停止开发环境
	@./scripts/deploy-dev.sh down

dev-restart: ## 重启开发环境
	@./scripts/deploy-dev.sh restart

dev-logs: ## 查看开发环境日志
	@./scripts/deploy-dev.sh logs

dev-shell: ## 进入后端容器
	docker exec -it itsm-backend-dev sh

dev-db-shell: ## 进入数据库容器
	docker exec -it itsm-postgres-dev psql -U itsm -d itsm

dev-doctor: ## 诊断开发环境问题
	@./scripts/deploy-dev.sh doctor

# ============================================
# 构建
# ============================================
build: build-backend build-frontend ## 构建所有镜像

build-backend: ## 构建后端镜像
	@echo "$(BLUE)构建后端镜像...$(NC)"
	docker build -t itsm-backend:dev -f $(BACKEND_DIR)/Dockerfile ./$(BACKEND_DIR)
	@echo "$(GREEN)后端镜像构建完成$(NC)"

build-frontend: ## 构建前端镜像
	@echo "$(BLUE)构建前端镜像...$(NC)"
	docker build -t itsm-frontend:dev -f $(FRONTEND_DIR)/Dockerfile --target production ./$(FRONTEND_DIR)
	@echo "$(GREEN)前端镜像构建完成$(NC)"

# ============================================
# 开箱即用（0→1）验收
# ============================================
oob-up: ## 启动开箱即用环境
	@echo "$(BLUE)启动开箱即用环境...$(NC)"
	docker compose -p itsm_oob -f docker-compose.yml up -d --build
	@echo "$(GREEN)环境已启动$(NC)"
	@echo "   前端: http://localhost:3000"
	@echo "   后端: http://localhost:8090"

oob-down: ## 停止并清理开箱即用环境
	@echo "$(YELLOW)停止并清理开箱即用环境...$(NC)"
	docker compose -p itsm_oob -f docker-compose.yml down -v
	@echo "$(GREEN)已清理$(NC)"

oob-test: ## 运行验收测试
	@echo "$(BLUE)运行验收测试...$(NC)"
	@./scripts/smoke-test.sh
	@echo "$(GREEN)验收通过$(NC)"

# ============================================
# 本地运行（不使用 Docker）
# ============================================
run: ## 启动后端服务（本地）
	cd $(BACKEND_DIR) && go run main.go

run-dev: ## 启动后端（air 热重载）
	@if ! command -v air >/dev/null; then \
		echo "$(YELLOW)air 未安装，正在安装...$(NC)"; \
		go install github.com/air-verse/air@latest; \
	fi
	cd $(BACKEND_DIR) && air

frontend-run: ## 启动前端（本地）
	cd $(FRONTEND_DIR) && npm run dev

frontend-build: ## 构建前端（生产）
	cd $(FRONTEND_DIR) && npm run build

# ============================================
# 测试
# ============================================
test: test-backend test-frontend ## 运行所有测试

test-backend: ## 运行后端测试
	@echo "$(BLUE)运行后端测试...$(NC)"
	cd $(BACKEND_DIR) && go test ./... -v -count=1

test-backend-coverage: ## 后端测试覆盖率
	cd $(BACKEND_DIR) && go test ./... -coverprofile=coverage.out
	cd $(BACKEND_DIR) && go tool cover -html=coverage.out -o coverage.html

test-frontend: ## 运行前端测试
	@echo "$(BLUE)运行前端测试...$(NC)"
	cd $(FRONTEND_DIR) && npm test -- --watchAll=false

# ============================================
# 代码质量
# ============================================
lint: lint-backend lint-frontend ## 代码检查

lint-backend: ## 后端代码检查
	cd $(BACKEND_DIR) && go vet ./...
	cd $(BACKEND_DIR) && staticcheck ./...

lint-frontend: ## 前端代码检查
	cd $(FRONTEND_DIR) && npm run lint
	cd $(FRONTEND_DIR) && npm run type-check

fmt: fmt-backend fmt-frontend ## 代码格式化

fmt-backend: ## 格式化后端代码
	cd $(BACKEND_DIR) && gofmt -w .

fmt-frontend: ## 格式化前端代码
	cd $(FRONTEND_DIR) && npm run format

# ============================================
# 数据库
# ============================================
db-migrate: ## 运行数据库迁移
	cd $(BACKEND_DIR) && go run -tags migrate main.go

db-seed: ## 导入初始数据
	docker compose -f $(DOCKER_COMPOSE_DEV) --profile dev restart itsm-backend

db-backup: ## 备份数据库
	@mkdir -p ./backups
	docker compose -f $(DOCKER_COMPOSE_DEV) --profile dev exec -T postgres pg_dump -U itsm itsm > ./backups/itsm_dev_$$(date +%Y%m%d_%H%M%S).sql

# ============================================
# 生产环境
# ============================================
prod-up: ## 部署生产环境
	@./scripts/deploy-prod.sh deploy

prod-down: ## 停止生产环境
	@./scripts/deploy-prod.sh down

prod-logs: ## 查看生产环境日志
	@./scripts/deploy-prod.sh logs

prod-restart: ## 重启生产环境
	docker compose --env-file .env.prod -f $(DOCKER_COMPOSE_PROD) restart

prod-status: ## 生产环境状态
	@./scripts/deploy-prod.sh status

prod-health: ## 生产环境健康检查
	@./scripts/deploy-prod.sh health

prod-backup: ## 备份生产数据库
	@./scripts/deploy-prod.sh backup

prod-rollback: ## 回滚到上一个版本
	@./scripts/deploy-prod.sh rollback

prod-dry-run: ## 预览生产部署
	@./scripts/deploy-prod.sh deploy --dry-run

# ============================================
# 维护
# ============================================
clean: ## 清理构建产物和缓存
	@echo "$(YELLOW)清理缓存和构建产物...$(NC)"
	cd $(BACKEND_DIR) && rm -rf bin/ coverage.out coverage.html
	cd $(FRONTEND_DIR) && rm -rf .next/ coverage/ dist/ build/
	rm -rf .deploy/ .pids/
	@echo "$(GREEN)清理完成$(NC)"

health: ## 检查服务健康状态
	@./scripts/deploy-dev.sh health

# ============================================
# 工具
# ============================================
swagger: ## 生成 Swagger 文档
	cd $(BACKEND_DIR) && swag init -g cmd/main.go -o docs --parseDependency --parseInternal

install-tools: ## 安装开发工具
	cd $(BACKEND_DIR) && go mod download
	cd $(BACKEND_DIR) && go install github.com/swaggo/swag/cmd/swag@latest
	cd $(BACKEND_DIR) && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	cd $(BACKEND_DIR) && go install github.com/air-verse/air@latest
	cd $(BACKEND_DIR) && go install mvdan.cc/gofumpt@latest
	cd $(BACKEND_DIR) && go install honnef.co/go/tools/cmd/staticcheck@latest
	cd $(FRONTEND_DIR) && npm install

ci: lint test ## 模拟 CI 流程

# 快速开始
quickstart: ## 一键启动开发环境
	@./scripts/deploy-dev.sh init

# 统一入口
start: dev-up ## 启动开发环境（别名）
status: dev-doctor ## 查看状态（别名）
shutdown: dev-down ## 停止服务（别名）
