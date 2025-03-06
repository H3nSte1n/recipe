# make/variables.mk

# Application Variables
APP_NAME=recipe-app
MAIN_PACKAGE=cmd/api/main.go
BINARY_NAME=recipe-app
VERSION?=$(shell git describe --tags --always --dirty)

# Go Variables
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOGET=$(GOCMD) get
GOFMT=$(GOCMD) fmt

# Docker Variables
DOCKER_COMPOSE=docker-compose
DOCKER_BUILD=docker build

# Database Variables
DB_USER?=postgres
DB_PASSWORD?=your_password
DB_HOST?=db
DB_PORT?=5432
DB_NAME?=recipe_db
DB_URL=postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

DOCKER_COMPOSE=docker-compose
DOCKER_EXEC=$(DOCKER_COMPOSE) exec

# Build flags
LDFLAGS=-ldflags "-X main.Version=${VERSION}"