#!/bin/bash

set -e

echo "🚀 Go-CacheX Development Script"
echo "================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go 1.24+ first."
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.24"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    print_error "Go version $GO_VERSION is too old. Please install Go $REQUIRED_VERSION+"
    exit 1
fi

print_status "Go version: $GO_VERSION"

# Download dependencies
echo ""
echo "📦 Downloading dependencies..."
go mod download
print_status "Dependencies downloaded"

# Run tests
echo ""
echo "🧪 Running tests..."
if go test -race -v ./...; then
    print_status "All tests passed"
else
    print_error "Tests failed"
    exit 1
fi

# Run linting if golangci-lint is available
if command -v golangci-lint &> /dev/null; then
    echo ""
    echo "🔍 Running linting..."
    if golangci-lint run; then
        print_status "Linting passed"
    else
        print_warning "Linting failed (continuing anyway)"
    fi
else
    print_warning "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
fi

# Run security check if govulncheck is available
if command -v govulncheck &> /dev/null; then
    echo ""
    echo "🔒 Running security check..."
    if govulncheck ./...; then
        print_status "Security check passed"
    else
        print_warning "Security check found vulnerabilities"
    fi
else
    print_warning "govulncheck not found. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi

# Build examples
echo ""
echo "🔨 Building examples..."
if go build -o bin/basic-example ./example/basic; then
    print_status "Basic example built successfully"
else
    print_error "Failed to build basic example"
    exit 1
fi

# Check if Redis is running
echo ""
echo "🔍 Checking Redis connection..."
if command -v redis-cli &> /dev/null; then
    if redis-cli ping &> /dev/null; then
        print_status "Redis is running"
        
        echo ""
        echo "🎯 Running basic example..."
        if timeout 10s ./bin/basic-example; then
            print_status "Basic example completed successfully"
        else
            print_warning "Basic example failed or timed out"
        fi
    else
        print_warning "Redis is not running. Start with: docker run -d -p 6379:6379 redis:7-alpine"
    fi
else
    print_warning "redis-cli not found. Install Redis or use Docker."
fi

echo ""
echo "🎉 Development setup complete!"
echo ""
echo "Next steps:"
echo "  • Start Redis: docker run -d -p 6379:6379 redis:7-alpine"
echo "  • Run example: ./bin/basic-example"
echo "  • Use Docker Compose: docker-compose -f deployments/docker-compose.yml up -d"
echo "  • View metrics: http://localhost:9090 (Prometheus)"
echo "  • View dashboards: http://localhost:3000 (Grafana, admin/admin)"
