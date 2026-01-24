.PHONY: build build-frontend build-backend test clean dev run help

# Default target
all: build

# Build everything (frontend then backend)
build: build-frontend build-backend

# Build frontend and copy to internal/static/dist
build-frontend:
	@echo "Building frontend..."
	cd web && npm run build
	@echo "Copying frontend build to internal/static/dist..."
	rm -rf internal/static/dist
	cp -r web/dist internal/static/

# Build backend binary
build-backend:
	@echo "Building backend..."
	go build -o bin/ccloganalysis ./cmd/server

# Run tests
test: build-frontend
	@echo "Running tests..."
	go test ./... -v

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf web/dist/
	rm -rf internal/static/dist/

# Development mode (backend only with CORS enabled)
dev:
	@echo "Starting development server with CORS enabled..."
	ENABLE_CORS=true go run ./cmd/server/main.go

# Run the built binary
run: build
	@echo "Running ccloganalysis..."
	./bin/ccloganalysis

# Show help
help:
	@echo "Available targets:"
	@echo "  build            - Build both frontend and backend (default)"
	@echo "  build-frontend   - Build React frontend only"
	@echo "  build-backend    - Build Go backend only"
	@echo "  test             - Run all tests"
	@echo "  clean            - Remove build artifacts"
	@echo "  dev              - Run development server with CORS"
	@echo "  run              - Build and run the application"
	@echo "  help             - Show this help message"
