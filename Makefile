# Makefile for the Golang project

# Variables
BINARY_NAME=server
PACKAGE=cmd/server
DOCS_DIR=docs

.PHONY: all build run test clean docs push

# Default target to run when executing 'make'
all: build

# Build the project
build:
	@echo "Building..."
	go build -o $(BINARY_NAME) ./$(PACKAGE)

# Run the server
run:
	@echo "Running server..."
	go run ./$(PACKAGE)

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Remove build artifacts
clean:
	@echo "Cleaning up..."
	go clean
	rm -f $(BINARY_NAME)

push:
	@echo "Pushing to GitHub..."
	git add .
	@read -p "Enter commit message: " commit_msg; \
	git commit -m "$$commit_msg"; \
	git push