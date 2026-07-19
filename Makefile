# ITSM Makefile

SHELL := /bin/bash

# Development
dev-init:           ## First-time development setup
	./scripts/deploy-dev.sh init

dev-start:          ## Start development environment
	./scripts/deploy-dev.sh up

dev-stop:           ## Stop development environment
	./scripts/deploy-dev.sh down

dev-logs:           ## View development logs
	./scripts/deploy-dev.sh logs

dev-restart:        ## Restart development environment
	./scripts/deploy-dev.sh restart

dev-status:         ## Show service status
	./scripts/deploy-dev.sh status

dev-health:         ## Run development health checks
	./scripts/deploy-dev.sh health

dev-doctor:         ## Diagnose local development environment
	./scripts/deploy-dev.sh doctor

dev-clean:          ## Clean up dev environment (remove containers and volumes)
	./scripts/deploy-dev.sh reset

# Production
prod-init:          ## Create .env.prod with generated secrets
	./scripts/deploy-prod.sh init

prod-deploy:        ## Full production deploy (validate → backup → build → deploy → verify)
	./scripts/deploy-prod.sh deploy

prod-start:         ## Start production environment with existing images
	./scripts/deploy-prod.sh deploy --skip-build --skip-backup

prod-stop:          ## Stop production environment
	./scripts/deploy-prod.sh down

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

# Build images
build-images:      ## Build all service images (VERSION=... REGISTRY=... make build-images)
	./scripts/build-images.sh $(VERSION) $(REGISTRY)

# Database
db-migrate:         ## Run database migrations
	cd itsm-backend && go run -tags migrate main.go

db-seed:            ## Seed database with test data
	cd itsm-backend && go run -tags create_user main.go

# Utility
logs-backend:       ## View backend logs
	./scripts/deploy-dev.sh logs itsm-backend

logs-frontend:     ## View frontend logs
	./scripts/deploy-dev.sh logs itsm-frontend

logs-postgres:      ## View postgres logs
	./scripts/deploy-dev.sh logs postgres

check-contracts:    ## Validate cross-file API, deployment, Docker, and docs contracts
	node scripts/check-engineering-contracts.js
	node scripts/check-api-paths.js

.PHONY: dev-init dev-start dev-stop dev-logs dev-restart dev-status dev-health dev-doctor dev-clean \
        prod-init prod-start prod-stop prod-deploy prod-status prod-health prod-logs \
        prod-restart prod-rollback prod-backup prod-down \
        db-migrate db-seed \
        release build-images \
        logs-backend logs-frontend logs-postgres check-contracts
