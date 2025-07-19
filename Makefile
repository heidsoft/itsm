.PHONY: help build-backend build-frontend run-backend run-frontend dev clean lint-fix

help: ## Show this help message
	@echo "Available commands:"
	@echo "  build-backend   - Build the Go backend"
	@echo "  build-frontend  - Build the Next.js frontend"
	@echo "  run-backend     - Run the backend server"
	@echo "  run-frontend    - Run the frontend development server"
	@echo "  dev             - Run both frontend and backend in development mode"
	@echo "  clean           - Clean build artifacts"
	@echo "  lint-fix        - Fix linting issues in frontend"

build-backend: ## Build the Go backend
	@echo "Building backend..."
	cd itsm-backend && go build -o itsm-backend .

build-frontend: ## Build the Next.js frontend
	@echo "Building frontend..."
	cd itsm-prototype && npm run build

run-backend: ## Run the backend server
	@echo "Starting backend server..."
	cd itsm-backend && ./itsm-backend

run-frontend: ## Run the frontend development server
	@echo "Starting frontend development server..."
	cd itsm-prototype && npm run dev

dev: ## Run both frontend and backend in development mode
	@echo "Starting development environment..."
	@make -j 2 run-backend run-frontend

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -f itsm-backend/itsm-backend
	rm -rf itsm-prototype/.next
	rm -rf itsm-prototype/out

lint-fix: ## Fix linting issues in frontend
	@echo "Fixing linting issues..."
	cd itsm-prototype && node scripts/comprehensive-fix.js
	cd itsm-prototype && npm run lint -- --fix

install-deps: ## Install dependencies for both projects
	@echo "Installing backend dependencies..."
	cd itsm-backend && go mod tidy
	@echo "Installing frontend dependencies..."
	cd itsm-prototype && npm install

setup: install-deps build-backend ## Setup the entire project
	@echo "Setup completed!"

test-backend: ## Run backend tests
	@echo "Running backend tests..."
	cd itsm-backend && go test ./...

test-frontend: ## Run frontend tests
	@echo "Running frontend tests..."
	cd itsm-prototype && npm test

test: test-backend test-frontend ## Run all tests 