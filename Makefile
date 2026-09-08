.PHONY: up down build lint

up:
	docker compose up --build

down:
	docker compose down -v

build:
	go build ./...

lint:
	golangci-lint run ./...
