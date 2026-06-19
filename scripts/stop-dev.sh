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
echo -e "${BLUE}  ITSM Development Environment Stop${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if any containers exist
echo -e "${CYAN}[1/3] Checking running containers...${NC}"
if ! docker compose -f docker-compose.dev.yml ps --quiet 2>/dev/null | grep -q .; then
    echo -e "${YELLOW}No dev containers are running.${NC}"
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  Cleanup Complete${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    exit 0
fi

# Show current status
echo -e "${GREEN}Found running containers:${NC}"
docker compose -f docker-compose.dev.yml ps
echo ""

# 2. Ask about volume removal
echo -e "${CYAN}[2/3] Volume cleanup option:${NC}"
echo ""
echo -e "Volumes associated with dev environment:"
echo -e "  - postgres_dev_data"
echo -e "  - redis_dev_data"
echo -e "  - minio_dev_data"
echo -e "  - ollama_dev_data"
echo -e "  - prometheus_dev_data"
echo -e "  - grafana_dev_data"
echo ""

read -p "Do you want to remove volumes? This will delete all data. (y/N): " -n 1 -r
echo ""
echo ""

# 3. Stop services
echo -e "${CYAN}[3/3] Stopping services...${NC}"

if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}Stopping services and removing volumes...${NC}"
    docker compose -f docker-compose.dev.yml --profile dev down -v
    echo ""
    echo -e "${GREEN}Services stopped and volumes removed.${NC}"
else
    echo -e "${YELLOW}Stopping services (keeping volumes)...${NC}"
    docker compose -f docker-compose.dev.yml --profile dev down
    echo ""
    echo -e "${GREEN}Services stopped. Volumes preserved.${NC}"
    echo -e "${YELLOW}To remove volumes manually, run:${NC}"
    echo -e "  docker volume rm itsm_postgres_dev_data itsm_redis_dev_data itsm_minio_dev_data itsm_ollama_dev_data itsm_prometheus_dev_data itsm_grafana_dev_data"
fi

echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Cleanup Complete${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""