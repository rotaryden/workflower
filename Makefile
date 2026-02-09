.PHONY: build run dev clean deploy test

include .env
include .deploy.env

export

# Run the application
run: build
	@echo "🚀 Running..."
	./$(APP_NAME)

# Development mode with auto-reload (requires air)
dev:
	@echo "🔄 Starting development server..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Installing air..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	rm -f $(APP_NAME)

# Build for Linux
build:
	go mod tidy
	@echo "🐧 Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -o $(APP_NAME) .

# Deploy to remote server
deploy: build
	@echo "📦 Deploying..."
	./$(APP_NAME) -D

remote-status:
	@echo "🔍 Checking remote status..."
	ssh $(REMOTE_HOST) -p $(SSH_PORT) "sudo systemctl status $(APP_NAME)"

remote-logs:
	@echo "📜 Checking remote logs..."
	ssh $(REMOTE_HOST) -p $(SSH_PORT) "sudo journalctl -u $(APP_NAME) -f"

# Download dependencies
deps:
	@echo "📥 Downloading dependencies..."
	go mod download
	go mod tidy

# Format code
fmt:
	@echo "✨ Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "🔍 Linting..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi

