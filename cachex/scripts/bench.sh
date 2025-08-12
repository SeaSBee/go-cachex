#!/bin/bash

set -e

echo "🏃 Go-CacheX Benchmark Script"
echo "============================="

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

# Check if Redis is running
echo "🔍 Checking Redis connection..."
if command -v redis-cli &> /dev/null; then
    if redis-cli ping &> /dev/null; then
        print_status "Redis is running"
    else
        print_warning "Redis is not running. Starting Redis..."
        docker run -d -p 6379:6379 redis:7-alpine
        sleep 3
        if redis-cli ping &> /dev/null; then
            print_status "Redis started successfully"
        else
            print_error "Failed to start Redis"
            exit 1
        fi
    fi
else
    print_warning "redis-cli not found. Starting Redis with Docker..."
    docker run -d -p 6379:6379 redis:7-alpine
    sleep 3
fi

# Create benchmark directory
mkdir -p benchmark-results

echo ""
echo "🧪 Running benchmarks..."

# Run cache benchmarks
echo "Running cache operation benchmarks..."
go test -bench=Benchmark -benchmem ./cachex/pkg/cache/ -v > benchmark-results/cache-benchmarks.txt 2>&1

# Run Redis store benchmarks
echo "Running Redis store benchmarks..."
go test -bench=Benchmark -benchmem ./cachex/pkg/redisstore/ -v > benchmark-results/redis-benchmarks.txt 2>&1

# Run codec benchmarks
echo "Running codec benchmarks..."
go test -bench=Benchmark -benchmem ./cachex/pkg/codec/ -v > benchmark-results/codec-benchmarks.txt 2>&1

# Run key builder benchmarks
echo "Running key builder benchmarks..."
go test -bench=Benchmark -benchmem ./cachex/pkg/key/ -v > benchmark-results/key-benchmarks.txt 2>&1

echo ""
echo "📊 Benchmark Results Summary:"
echo "============================="

# Display cache benchmarks
if [ -f benchmark-results/cache-benchmarks.txt ]; then
    echo ""
    echo "Cache Operations:"
    echo "----------------"
    grep -E "Benchmark.*op" benchmark-results/cache-benchmarks.txt || echo "No cache benchmarks found"
fi

# Display Redis benchmarks
if [ -f benchmark-results/redis-benchmarks.txt ]; then
    echo ""
    echo "Redis Store Operations:"
    echo "----------------------"
    grep -E "Benchmark.*op" benchmark-results/redis-benchmarks.txt || echo "No Redis benchmarks found"
fi

# Display codec benchmarks
if [ -f benchmark-results/codec-benchmarks.txt ]; then
    echo ""
    echo "Codec Operations:"
    echo "----------------"
    grep -E "Benchmark.*op" benchmark-results/codec-benchmarks.txt || echo "No codec benchmarks found"
fi

# Display key builder benchmarks
if [ -f benchmark-results/key-benchmarks.txt ]; then
    echo ""
    echo "Key Builder Operations:"
    echo "----------------------"
    grep -E "Benchmark.*op" benchmark-results/key-benchmarks.txt || echo "No key builder benchmarks found"
fi

echo ""
print_status "Benchmarks completed successfully!"
echo ""
echo "📁 Detailed results saved in benchmark-results/ directory"
echo ""
echo "To view detailed results:"
echo "  • cat benchmark-results/cache-benchmarks.txt"
echo "  • cat benchmark-results/redis-benchmarks.txt"
echo "  • cat benchmark-results/codec-benchmarks.txt"
echo "  • cat benchmark-results/key-benchmarks.txt"
