# Makefile
.PHONY: help build up down logs clean init-key

help:
	@echo "Available commands:"
	@echo "  make build      - Build Docker images"
	@echo "  make up         - Start all services"
	@echo "  make down       - Stop all services"
	@echo "  make logs       - Show logs"
	@echo "  make clean      - Remove containers and volumes"
	@echo "  make init-key   - Generate initial API key"
	@echo "  make restart    - Restart all services"
	@echo "  make ps         - Show running containers"

build:
	docker compose build

up:
	docker compose up -d
	@echo "Waiting for services to start..."
	@sleep 10
	docker compose ps
	@echo "API is available at http://localhost:8080"

down:
	docker compose down

logs:
	docker compose logs -f

clean:
	docker compose down -v
	rm -rf data/*

init-key:
	docker compose exec api ./generate-key

restart:
	docker compose restart

ps:
	docker compose ps

db-mysql:
	docker compose exec mysql mysql -u root -ppassword ipset

db-postgres:
	docker compose exec postgres psql -U ipset_user -d ipset