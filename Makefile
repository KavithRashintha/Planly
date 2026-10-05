.PHONY: up down build test lint tidy sync-work

up:
	docker compose up -d

down:
	docker compose down

build:
	docker compose build

test:
	go test -v ./pkg/... ./services/gateway/... ./services/auth/... ./services/planner/... ./services/notify/... ./services/agent/...

lint:
	go vet ./pkg/... ./services/gateway/... ./services/auth/... ./services/planner/... ./services/notify/... ./services/agent/...

tidy:
	cd pkg && go mod tidy
	cd services/gateway && go mod tidy
	cd services/auth && go mod tidy
	cd services/planner && go mod tidy
	cd services/notify && go mod tidy
	cd services/agent && go mod tidy
	go work sync

sync-work:
	go work sync
