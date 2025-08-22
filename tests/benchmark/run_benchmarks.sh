#!/bin/bash

# Benchmark test runner for go-cachex
# This script runs all benchmark tests and provides a summary

set -e

echo "🚀 Starting go-cachex Benchmark Tests"
echo "====================================="

# Check if Redis is running
echo "📋 Checking Redis server availability..."
if ! redis-cli ping > /dev/null 2>&1; then
    echo "❌ Redis server is not running. Please start Redis at localhost:6379"
    echo "   You can start Redis with: redis-server"
    exit 1
fi
echo "✅ Redis server is running"

# Create benchmark results directory
mkdir -p benchmark_results

# Run all benchmarks with detailed output
echo ""
echo "🏃 Running benchmark tests..."
echo "============================"

# Run benchmarks and capture output
go test -v -bench=. -benchmem ./tests/benchmark/ 2>&1 | tee benchmark_results/benchmark_output.txt

# Extract and format results
echo ""
echo "📊 Benchmark Results Summary"
echo "============================"

# Extract benchmark results and format them
grep -E "^Benchmark" benchmark_results/benchmark_output.txt | while read line; do
    # Parse the benchmark line
    name=$(echo "$line" | awk '{print $1}')
    ops=$(echo "$line" | awk '{print $3}')
    ns_per_op=$(echo "$line" | awk '{print $4}')
    bytes_per_op=$(echo "$line" | awk '{print $5}')
    allocs_per_op=$(echo "$line" | awk '{print $6}')
    
    printf "%-30s %8s ops %12s %12s %12s\n" "$name" "$ops" "$ns_per_op" "$bytes_per_op" "$allocs_per_op"
done

echo ""
echo "📁 Detailed results saved to: benchmark_results/benchmark_output.txt"
echo ""
echo "🎯 Benchmark Categories Covered:"
echo "  • Redis Store Operations (Set, Get, MSet, MGet, IncrBy)"
echo "  • Memory Store Operations (Set, Get, MSet, MGet)"
echo "  • Ristretto Store Operations (Set, Get)"
echo "  • Layered Store Operations (Set, Get)"
echo "  • Tagging Operations (AddTags, GetKeysByTag)"
echo "  • Codec Operations (MessagePack vs JSON Encode/Decode)"
echo "  • Cache Configurations (Redis, Memory, Layered)"
echo "  • Complex Data Structures"
echo "  • Large Data Operations"
echo "  • TTL Operations"
echo "  • Concurrent Operations"
echo "  • Key Builder Operations"
echo "  • Observability Operations"
echo "  • Configuration Creation"

echo ""
echo "✅ Benchmark tests completed successfully!"
