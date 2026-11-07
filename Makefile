.PHONY: build run dev docker-build docker-run clean test deps

PORT ?= 8080
DOMAIN ?= localhost

deps:
	go mod tidy
	go mod verify

build: deps
	go build -o gopress ./cmd/server

run:
	PORT=$(PORT) DOMAIN=$(DOMAIN) go run ./cmd/server

dev: deps
	PORT=$(PORT) DOMAIN=$(DOMAIN) go run ./cmd/server

test:
	go test -v ./...

docker-build:
	docker-compose build

docker-run:
	docker-compose up

docker-dev:
	docker-compose up --build -d

docker-down:
	docker-compose down

clean:
	rm -f gopress
	rm -f *.db

seed:
	go run ./cmd/server -seed-only
