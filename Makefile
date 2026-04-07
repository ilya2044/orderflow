.PHONY: up down build logs restart clean migrate-up migrate-down proto test lint seed help

SERVICES = api-gateway auth-service user-service product-service order-service inventory-service payment-service notification-service
GO_VERSION = 1.22

help:
	@echo "Order Processing Microservices"
	@echo ""
	@echo "Usage:"
	@echo "  make up          - Start all services"
	@echo "  make down        - Stop all services"
	@echo "  make build       - Build all Docker images"
	@echo "  make logs        - Follow all logs"
	@echo "  make restart     - Restart all services"
	@echo "  make clean       - Remove containers, volumes, images"
	@echo "  make migrate-up  - Run all migrations"
	@echo "  make proto       - Generate gRPC code from proto files"
	@echo "  make test        - Run all tests"
	@echo "  make lint        - Run linter"
	@echo "  make seed        - Seed database with sample data"
	@echo "  make infra       - Start only infrastructure (DB, Kafka, Redis, etc.)"
	@echo ""

up:
	docker compose up -d

down:
	docker compose down

build:
	docker compose build --parallel

logs:
	docker compose logs -f

restart:
	docker compose restart

clean:
	docker compose down -v --rmi all --remove-orphans

infra:
	docker compose up -d postgres mongo redis zookeeper kafka elasticsearch minio jaeger prometheus grafana

infra-down:
	docker compose stop postgres mongo redis zookeeper kafka elasticsearch minio jaeger prometheus grafana

migrate-up:
	@for service in auth-service user-service order-service inventory-service payment-service; do \
		echo "Running migrations for $$service..."; \
		cd $$service && go run cmd/migrate/main.go up && cd ..; \
	done

migrate-down:
	@for service in auth-service user-service order-service inventory-service payment-service; do \
		echo "Rolling back migrations for $$service..."; \
		cd $$service && go run cmd/migrate/main.go down && cd ..; \
	done

proto:
	@which protoc > /dev/null || (echo "protoc not found. Install: https://grpc.io/docs/protoc-installation/" && exit 1)
	@which protoc-gen-go > /dev/null || go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@which protoc-gen-go-grpc > /dev/null || go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		auth-service/proto/auth.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		inventory-service/proto/inventory.proto

test:
	@for service in $(SERVICES); do \
		echo "Testing $$service..."; \
		cd $$service && go test ./... -v -race && cd ..; \
	done

lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@for service in $(SERVICES); do \
		echo "Linting $$service..."; \
		cd $$service && golangci-lint run && cd ..; \
	done

seed:
	go run scripts/seed/main.go

work-sync:
	go work sync

tidy:
	@for service in $(SERVICES) pkg; do \
		echo "Tidying $$service..."; \
		cd $$service && go mod tidy && cd ..; \
	done

ps:
	docker compose ps

kafka-topics:
	docker exec order_kafka kafka-topics --list --bootstrap-server localhost:9092

kafka-ui:
	open http://localhost:8090

grafana:
	open http://localhost:3001

jaeger:
	open http://localhost:16686

minio:
	open http://localhost:9001

frontend:
	open http://localhost:3000

api:
	open http://localhost:8080/swagger/index.html
