#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  ITSM Development Environment Startup${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 1. Check if Docker is running
echo -e "${CYAN}[1/4] Checking Docker status...${NC}"
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}ERROR: Docker is not running or not accessible.${NC}"
    echo -e "${YELLOW}Please start Docker Desktop or Docker Engine and try again.${NC}"
    exit 1
fi
echo -e "${GREEN}Docker is running.${NC}"

# 2. Create required directories if they don't exist
echo -e "${CYAN}[2/4] Creating required directories...${NC}"
for dir in logs uploads; do
    if [ ! -d "$dir" ]; then
        mkdir -p "$dir"
        echo -e "${GREEN}Created directory: $dir/${NC}"
    else
        echo -e "${YELLOW}Directory already exists: $dir/${NC}"
    fi
done

# 3. Start services
echo -e "${CYAN}[3/4] Starting services...${NC}"
echo -e "${YELLOW}This may take a few minutes on first run (building images)...${NC}"
echo ""

# Cleanup function for Ctrl+C
cleanup() {
    echo ""
    echo -e "${YELLOW}Received interrupt signal. Stopping services...${NC}"
    docker compose -f docker-compose.dev.yml --profile dev down
    echo -e "${GREEN}Services stopped.${NC}"
    exit 0
}

# Set trap for Ctrl+C
trap cleanup SIGINT SIGTERM

# Start services in detached mode
docker compose -f docker-compose.dev.yml --profile dev up -d

echo ""
echo -e "${GREEN}Services started successfully!${NC}"

# 4. Show service status and access URLs
echo -e "${CYAN}[4/4] Service Status:${NC}"
echo ""

# Wait a moment for containers to stabilize
sleep 2

# Show container status
docker compose -f docker-compose.dev.yml ps

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Access URLs${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${GREEN}Frontend:${NC}         http://localhost:3000"
echo -e "${GREEN}Backend API:${NC}       http://localhost:8090"
echo -e "${GREEN}Swagger Docs:${NC}      http://localhost:8090/swagger/index.html"
echo -e "${GREEN}PostgreSQL:${NC}        localhost:5432"
echo -e "${GREEN}Redis:${NC}             localhost:6379"
echo -e "${GREEN}MinIO Console:${NC}     http://localhost:9001"
echo -e "${GREEN}Ollama:${NC}            http://localhost:11434"
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop all services${NC}"
echo ""

# Wait for services to be healthy
echo -e "${CYAN}Waiting for services to become healthy...${NC}"

# Function to check if a container is healthy
check_healthy() {
    local container=$1
    local status=$(docker inspect --format='{{.State.Health.Status}}' "$container" 2>/dev/null || echo "none")
    if [ "$status" = "healthy" ]; then
        return 0
    fi
    return 1
}

# Wait for backend to be healthy (max 60 seconds)
backend_status=""
for i in {1..60}; do
    backend_status=$(docker inspect --format='{{.State.Health.Status}}' "itsm-backend-dev" 2>/dev/null || echo "unknown")
    if [ "$backend_status" = "healthy" ]; then
        echo -e "${GREEN}Backend is healthy!${NC}"
        break
    fi
    echo -e "${YELLOW}Waiting for backend... (${i}s) Status: ${backend_status}${NC}"
    sleep 1
done

if [ "$backend_status" != "healthy" ]; then
    echo -e "${YELLOW}Warning: Backend may not be fully healthy yet.${NC}"
    echo -e "${YELLOW}Check logs with: docker compose -f docker-compose.dev.yml logs backend${NC}"
fi

echo ""
echo -e "${BLUE}Development environment is ready!${NC}"
echo -e "${YELLOW}View logs with: docker compose -f docker-compose.dev.yml logs -f${NC}"
echo ""

# Keep script running to handle Ctrl+C
wait
