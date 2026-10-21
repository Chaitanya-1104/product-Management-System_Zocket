#!/bin/bash

# Quick Start Guide for Product Management System

set -e

echo "🚀 Starting Product Management System Setup..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed. Please install Docker first.${NC}"
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose is not installed. Please install Docker Compose first.${NC}"
    exit 1
fi

echo -e "${BLUE}📦 Building and starting services...${NC}"
docker-compose up --build -d

echo -e "${BLUE}⏳ Waiting for services to be ready...${NC}"
sleep 10

# Check if API is responding
echo -e "${BLUE}🔍 Verifying API service...${NC}"
if curl -f http://localhost:8080/health > /dev/null 2>&1; then
    echo -e "${GREEN}✅ API is running at http://localhost:8080${NC}"
else
    echo -e "${RED}❌ API failed to start. Checking logs...${NC}"
    docker-compose logs api
    exit 1
fi

# Create database schema
echo -e "${BLUE}🗄️  Setting up database...${NC}"
docker-compose exec -T database psql -U postgres -d product_management -f /docker-entrypoint-initdb.d/001_initial_schema.sql 2>/dev/null || \
    docker exec $(docker-compose ps -q database) psql -U postgres -d product_management < migrations/001_initial_schema.sql 2>/dev/null || \
    echo -e "${BLUE}⚠️  Database might already be initialized${NC}"

echo ""
echo -e "${GREEN}✅ Setup Complete!${NC}"
echo ""
echo -e "${BLUE}📍 Service Endpoints:${NC}"
echo "   API:              http://localhost:8080"
echo "   RabbitMQ UI:      http://localhost:15672 (guest/guest)"
echo "   PostgreSQL:       localhost:5432"
echo "   Redis:            localhost:6379"
echo ""
echo -e "${BLUE}📖 Quick Commands:${NC}"
echo "   View logs:        docker-compose logs -f api"
echo "   Worker logs:      docker-compose logs -f worker"
echo "   Stop services:    docker-compose down"
echo "   Restart services: docker-compose restart"
echo ""
echo -e "${BLUE}🧪 Test the API:${NC}"
echo "   1. Create a product:"
echo "      curl -X POST http://localhost:8080/products \\"
echo "        -H 'Content-Type: application/json' \\"
echo "        -d '{\"user_id\":1,\"product_name\":\"Test\",\"product_description\":\"Test\",\"product_images\":[\"https://via.placeholder.com/300\"],\"product_price\":99.99}'"
echo ""
echo "   2. Get product (wait a few seconds for worker to process):"
echo "      curl http://localhost:8080/products/1"
echo ""
