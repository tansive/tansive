# Go parameters
GO := go
GOBUILD := $(GO) build
GOCLEAN := $(GO) clean
GOTEST := $(GO) test

# Build directory
BUILDDIR := build

# Applications
APPS := tansivesrv tangent tansive

# Docker configuration
DOCKER_IMAGE_NAME := tansive
DOCKER_TAG := $(shell whoami)-latest

# Targets
.PHONY: all clean test build cli srv worker docker-build docker-build-multiarch docker-build-local docker-test-multiarch

all: build

build: $(APPS)

# Server target
srv:
	@echo "Building tansivesrv..."
	@mkdir -p $(BUILDDIR)
	$(GOBUILD) -o $(BUILDDIR)/tansivesrv ./cmd/tansivesrv

# Worker target
tangent:
	@echo "Building tangent..."
	@mkdir -p $(BUILDDIR)
	$(GOBUILD) -o $(BUILDDIR)/tangent ./cmd/tangent

# CLI target
cli:
	@echo "Building tansive..."
	@mkdir -p $(BUILDDIR)
	$(GOBUILD) -o $(BUILDDIR)/tansive ./cmd/tansive-cli

# Original targets for backward compatibility
tansivesrv: srv

tansive: cli

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILDDIR)

test:
	@echo "Running tests..."
	$(GOTEST) -p=1 -count=1 ./...
