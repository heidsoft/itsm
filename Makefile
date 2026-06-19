# ITSM Makefile

# Development
dev-start:          ## Start development environment (docker compose up)
	docker compose -f docker-compose.dev.yml up -d

dev-stop:           ## Stop development environment (docker compose down)
	docker compose -f docker-compose.dev.yml down

dev-logs:           ## View development logs (docker compose logs)
	docker compose -f docker-compose.dev.yml logs

dev-restart:        ## Restart development environment
	docker compose -f docker-compose.dev.yml down && docker compose -f docker-compose.dev.yml up -d

dev-status:         ## Show service status
	docker compose -f docker-compose.dev.yml ps

dev-clean:          ## Clean up dev environment (remove containers and volumes)
	docker compose -f docker-compose.dev.yml down -v

# Production
prod-deploy:        ## Full production deploy (validate → backup → build → deploy → verify)
	./scripts/deploy-prod.sh deploy

prod-start:         ## Start production environment (infrastructure only)
	docker compose -f docker-compose.prod.yml --env-file .env.prod up -d postgres redis minio

prod-stop:          ## Stop production environment
	docker compose -f docker-compose.prod.yml --env-file .env.prod down

prod-restart:       ## Restart production environment
	./scripts/deploy-prod.sh down && ./scripts/deploy-prod.sh deploy

prod-status:        ## Show production service status
	./scripts/deploy-prod.sh status

prod-health:        ## Run production health checks
	./scripts/deploy-prod.sh health

prod-logs:          ## View production logs
	./scripts/deploy-prod.sh logs

prod-rollback:      ## Rollback to previous deployment
	./scripts/deploy-prod.sh rollback

prod-backup:       ## Backup production database
	./scripts/deploy-prod.sh backup

prod-down:          ## Stop all production services
	./scripts/deploy-prod.sh down

# Release
release:            ## Create release artifacts (VERSION=v1.0.0 make release)
ifndef VERSION
	@echo "Usage: VERSION=v1.0.0 make release"
	@echo "Example: VERSION=v1.0.0 make release"
	@exit 1
endif
	./scripts/release.sh $(VERSION)

# Database
db-migrate:         ## Run database migrations
	cd itsm-backend && go run -tags migrate main.go

db-seed:            ## Seed database with test data
	cd itsm-backend && go run -tags create_user main.go

# Utility
logs-backend:       ## View backend logs
	docker compose -f docker-compose.dev.yml logs itsm-backend

logs-frontend:     ## View frontend logs
	docker compose -f docker-compose.dev.yml logs itsm-frontend

logs-postgres:      ## View postgres logs
	docker compose -f docker-compose.dev.yml logs postgres

.PHONY: dev-start dev-stop dev-logs dev-restart dev-status dev-clean \
        prod-start prod-stop prod-deploy prod-status prod-health prod-logs \
        prod-restart prod-rollback prod-backup prod-down \
        db-migrate db-seed \
        release \
        logs-backend logs-frontend logs-postgres
