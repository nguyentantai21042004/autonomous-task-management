.PHONY: help run-api run-consumer build-api build-consumer docker-build-api docker-build-consumer test clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

swagger:
	@echo "Generating swagger"
	@swag init -g cmd/api/main.go
	@echo "Fixing swagger docs (removing deprecated LeftDelim/RightDelim)..."
	@sed -i '' '/LeftDelim:/d' docs/docs.go
	@sed -i '' '/RightDelim:/d' docs/docs.go

run-api:
	@echo "Generating swagger"
	@swag init -g cmd/api/main.go
	@sed -i '' '/LeftDelim:/d' docs/docs.go
	@sed -i '' '/RightDelim:/d' docs/docs.go
	@echo "Running the application"
	@go run cmd/api/main.go

up: ## Start infrastructure via docker-compose
	docker compose --env-file .env -f manifests/docker-compose/docker-compose.yml up -d

down: ## Stop infrastructure via docker-compose
	docker compose --env-file .env -f manifests/docker-compose/docker-compose.yml down

logs: ## Tail backend logs
	docker compose --env-file .env -f manifests/docker-compose/docker-compose.yml logs -f backend