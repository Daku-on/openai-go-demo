# Go Research Assistant - Full Stack Build System
.PHONY: help build-all build-cli build-web build-wasm serve-web serve-wasm clean deps test

# Default target
help:
	@echo "🚀 Go Research Assistant - Build Commands"
	@echo "========================================"
	@echo ""
	@echo "📦 Build Commands:"
	@echo "  make build-all    - Build all versions (CLI, Web, WASM)"
	@echo "  make build-cli    - Build CLI version"
	@echo "  make build-web    - Build Web server"
	@echo "  make build-wasm   - Build WebAssembly version"
	@echo ""
	@echo "🌐 Serve Commands:"
	@echo "  make serve-web    - Start Web server (http://localhost:8080)"
	@echo "  make serve-wasm   - Serve WASM version (http://localhost:3000)"
	@echo ""
	@echo "🛠️  Utility Commands:"
	@echo "  make deps         - Download dependencies"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build artifacts"

# Install dependencies
deps:
	@echo "📥 Downloading dependencies..."
	@go mod download
	@go mod tidy

# Build all versions
build-all: build-cli build-web build-wasm
	@echo "✅ All builds completed!"

# Build CLI version
build-cli: deps
	@echo "🖥️  Building CLI version..."
	@go build -o bin/research-cli ./cmd/cli
	@echo "✅ CLI built: ./bin/research-cli"

# Build Web server
build-web: deps
	@echo "🌐 Building Web server..."
	@go build -o bin/research-web ./cmd/web
	@echo "✅ Web server built: ./bin/research-web"

# Build WebAssembly version
build-wasm: deps
	@echo "⚡ Building WebAssembly version..."
	@mkdir -p wasm/dist
	@GOOS=js GOARCH=wasm go build -o wasm/dist/main.wasm ./cmd/wasm
	@cp "$(shell go env GOROOT)/lib/wasm/wasm_exec.js" wasm/dist/
	@cp wasm/index.html wasm/dist/
	@echo "✅ WASM built: ./wasm/dist/"

# Serve Web version
serve-web: build-web
	@echo "🌐 Starting Web server on http://localhost:8080"
	@echo "   Graph Visualizer: http://localhost:8080"
	@cd cmd/web && ../../bin/research-web

# Serve WASM version
serve-wasm: build-wasm
	@echo "⚡ Starting WASM server on http://localhost:3000"
	@echo "   WASM App: http://localhost:3000"
	@cd wasm/dist && python3 -m http.server 3000 2>/dev/null || python -m SimpleHTTPServer 3000

# Run CLI version
run-cli: build-cli
	@echo "🖥️  Running CLI version..."
	@./bin/research-cli

# Test
test:
	@echo "🧪 Running tests..."
	@go test ./...

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf wasm/dist/
	@echo "✅ Clean completed"

# Quick demo setup
demo: build-all
	@echo "🎯 Demo Setup Complete!"
	@echo ""
	@echo "🎮 Choose your interface:"
	@echo "  1. CLI:  make run-cli"
	@echo "  2. Web:  make serve-web  (then visit http://localhost:8080)"
	@echo "  3. WASM: make serve-wasm (then visit http://localhost:3000)"
	@echo ""
	@echo "💡 Don't forget to set up your .env file with API keys!"

# Development mode - rebuild and serve web on file changes
dev-web:
	@echo "🔄 Development mode - Web (auto-rebuild)"
	@while true; do \
		make build-web; \
		echo "🌐 Starting dev server..."; \
		(cd cmd/web && ../../bin/research-web) & \
		SERVER_PID=$$!; \
		echo "⏳ Watching for changes... (Ctrl+C to stop)"; \
		inotifywait -e modify -r . 2>/dev/null || fswatch -o . | head -1; \
		kill $$SERVER_PID 2>/dev/null; \
		sleep 1; \
	done

# Check environment
check-env:
	@echo "🔍 Environment Check:"
	@echo "  Go version: $(shell go version)"
	@echo "  Project root: $(shell pwd)"
	@echo "  Dependencies:"
	@go list -m all | head -10
	@echo ""
	@if [ -f .env ]; then \
		echo "✅ .env file found"; \
	else \
		echo "⚠️  .env file not found - copy .env.example to .env"; \
	fi

# Performance build (optimized)
build-prod: deps
	@echo "🚀 Building production versions..."
	@mkdir -p bin
	@echo "Building CLI (optimized)..."
	@CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/research-cli ./cmd/cli
	@echo "Building Web server (optimized)..."
	@CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/research-web ./cmd/web
	@echo "Building WASM (optimized)..."
	@mkdir -p wasm/dist
	@GOOS=js GOARCH=wasm go build -ldflags="-w -s" -o wasm/dist/main.wasm ./cmd/wasm
	@cp "$(shell go env GOROOT)/lib/wasm/wasm_exec.js" wasm/dist/
	@cp wasm/index.html wasm/dist/
	@echo "✅ Production builds completed!"

# Show project structure
structure:
	@echo "📁 Project Structure:"
	@echo "====================="
	@tree -I 'node_modules|*.log|bin|dist' . || find . -type f -name "*.go" -o -name "*.html" -o -name "*.md" | sort