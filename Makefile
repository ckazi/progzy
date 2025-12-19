.PHONY: help up down restart logs clean build dev-backend dev-frontend

help:
	@echo "Available commands:"
	@echo "  make up          - Start all services"
	@echo "  make down        - Stop all services"
	@echo "  make restart     - Restart all services"
	@echo "  make logs        - View logs from all services"
	@echo "  make clean       - Stop services and remove volumes"
	@echo "  make build       - Rebuild all Docker images"
	@echo "  make dev-backend - Run backend in development mode"
	@echo "  make dev-frontend- Run frontend in development mode"

up:
	docker-compose up -d

down:
	docker-compose down

restart:
	docker-compose restart

logs:
	docker-compose logs -f

logs-backend:
	docker-compose logs -f backend

logs-frontend:
	docker-compose logs -f frontend

logs-db:
	docker-compose logs -f postgres

clean:
	docker-compose down -v

build:
	docker-compose build --no-cache

dev-backend:
	cd backend && go run main.go

dev-frontend:
	cd frontend && npm run dev
