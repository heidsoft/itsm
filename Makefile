# ITSM Makefile

SHELL := /bin/bash
VERSION ?= latest
REGISTRY ?=

# Development
dev-init:           ## First-time development setup
	./scripts/deploy-dev.sh init

dev-start:          ## Start development environment
	./scripts/deploy-dev.sh up

dev-start-local:    ## Start local Go/Next.js processes with Docker infrastructure
	./scripts/deploy-dev.sh up --local

dev-start-docker:   ## Start the Docker Compose development environment
	./scripts/deploy-dev.sh up --docker

dev-stop:           ## Stop development environment
	./scripts/deploy-dev.sh down

dev-stop-local:     ## Stop local Go/Next.js development processes
	./scripts/deploy-dev.sh down --local

dev-stop-docker:    ## Stop the Docker Compose development environment
	./scripts/deploy-dev.sh down --docker

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
	./scripts/build-images.sh "$(VERSION)" "$(REGISTRY)"

build-backend:     ## Build the backend image only
	./scripts/build-images.sh "$(VERSION)" "$(REGISTRY)" backend

build-frontend:    ## Build the frontend image only
	./scripts/build-images.sh "$(VERSION)" "$(REGISTRY)" frontend

verify-scripts:    ## Validate build/start scripts without starting services
	bash -n scripts/build-images.sh scripts/deploy-dev.sh scripts/deploy-prod.sh scripts/lib/common.sh
	node --test scripts/__tests__/build-start-scripts.test.js

# Database
db-migrate:         ## Show how schema changes are applied (safe, read-only)
	@echo "Schema is applied automatically when the backend starts (ITSM_AUTO_MIGRATE=true)."
	@echo ""
	@echo "  Apply schema normally :  make dev-start-docker"
	@echo "  Rebuild empty database:  make db-reset   (DESTRUCTIVE)"

db-reset:           ## DESTRUCTIVE: drop and recreate the database (was db-migrate)
	@echo ""
	@echo "  #############################################################"
	@echo "  #  WARNING: This DROPS the database and recreates it empty.  #"
	@echo "  #  ALL DATA WILL BE PERMANENTLY LOST.                        #"
	@echo "  #############################################################"
	@echo ""
	@if [ "$${DB_RESET_CONFIRM:-}" != "reset" ]; then \
		printf "Type 'reset' to continue: "; \
		read ans; \
		if [ "$$ans" != "reset" ]; then echo "Aborted."; exit 1; fi; \
	fi
	cd itsm-backend && $(MAKE) build && GOTOOLCHAIN="${GOTOOLCHAIN:-auto}" go run -tags migrate main.go

db-seed:            ## Seed database with test data
	cd itsm-backend && GOTOOLCHAIN="${GOTOOLCHAIN:-auto}" go run -tags create_user main.go

# Backend Go wrappers (delegates to itsm-backend/Makefile, keeps GOTOOLCHAIN=auto)
backend-test:       ## Run Go tests with toolchain auto (delegates to itsm-backend/Makefile)
	cd itsm-backend && $(MAKE) test

backend-test-ci:    ## Run Go tests with coverage (mirrors backend-ci.yml)
	cd itsm-backend && $(MAKE) test-ci

backend-vet:        ## Run `go vet ./...`
	cd itsm-backend && $(MAKE) vet

backend-build:      ## Build backend binary into itsm-backend/itsm
	cd itsm-backend && $(MAKE) build

backend-cover:      ## Generate coverage profile (itsm-backend/coverage.out)
	cd itsm-backend && $(MAKE) cover

backend-cover-html: ## Render coverage as HTML (itsm-backend/coverage.html)
	cd itsm-backend && $(MAKE) cover-html

backend-lint:       ## Run staticcheck locally (matches CI)
	cd itsm-backend && $(MAKE) lint

backend-tidy:       ## Run `go mod tidy` in itsm-backend/
	cd itsm-backend && $(MAKE) tidy

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

coverage-report:    ## Unified coverage report (Go + Jest) → coverage-summary.md
	./scripts/coverage-report.sh

coverage-baseline:  ## Snapshot current coverage to docs/testing/coverage-baseline.json
	mkdir -p docs/testing
	node -e "const fs=require('fs');const o={go:0,jest:0,ts:new Date().toISOString()};fs.writeFileSync('docs/testing/coverage-baseline.json',JSON.stringify(o,null,2));console.log('baseline (0,0) written; will be overwritten on next run')"

.PHONY: dev-init dev-start dev-start-local dev-start-docker dev-stop dev-stop-local dev-stop-docker dev-logs dev-restart dev-status dev-health dev-doctor dev-clean \
        prod-init prod-start prod-stop prod-deploy prod-status prod-health prod-logs \
        prod-restart prod-rollback prod-backup prod-down \
        db-migrate db-reset db-seed \
        backend-test backend-test-ci backend-vet backend-build backend-cover backend-cover-html backend-lint backend-tidy \
        release build-images build-backend build-frontend verify-scripts \
        logs-backend logs-frontend logs-postgres check-contracts coverage-report coverage-baseline
