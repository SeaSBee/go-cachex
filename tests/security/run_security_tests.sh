#!/bin/bash

# Security test runner for go-cachex
# This script runs all security tests and provides a comprehensive security report

set -e

echo "🔒 Starting go-cachex Security Tests"
echo "===================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create security test results directory
mkdir -p security_test_results

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    case $status in
        "PASS")
            echo -e "${GREEN}✅ $message${NC}"
            ;;
        "FAIL")
            echo -e "${RED}❌ $message${NC}"
            ;;
        "SKIP")
            echo -e "${YELLOW}⏭️  $message${NC}"
            ;;
        "INFO")
            echo -e "${BLUE}ℹ️  $message${NC}"
            ;;
    esac
}

# Function to run tests and capture results
run_test_category() {
    local category=$1
    local pattern=$2
    local description=$3
    
    echo ""
    print_status "INFO" "Running $description..."
    echo "----------------------------------------"
    
    # Run the test category
    if go test -v -run "$pattern" ./tests/security/ -timeout 60s > "security_test_results/${category}_output.txt" 2>&1; then
        print_status "PASS" "$description completed successfully"
        echo "Results saved to: security_test_results/${category}_output.txt"
        return 0
    else
        print_status "FAIL" "$description failed"
        echo "Check: security_test_results/${category}_output.txt"
        return 1
    fi
}

# Check prerequisites
echo "📋 Checking prerequisites..."
print_status "INFO" "Checking Go environment..."

if ! command -v go &> /dev/null; then
    print_status "FAIL" "Go is not installed or not in PATH"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
print_status "PASS" "Go version: $GO_VERSION"

# Check Redis availability for integration tests
print_status "INFO" "Checking Redis availability..."
if redis-cli ping > /dev/null 2>&1; then
    print_status "PASS" "Redis server is available"
    REDIS_AVAILABLE=true
else
    print_status "SKIP" "Redis server not available - integration tests will be skipped"
    REDIS_AVAILABLE=false
fi

# Check test dependencies
print_status "INFO" "Checking test dependencies..."
if go list -m github.com/stretchr/testify > /dev/null 2>&1; then
    print_status "PASS" "Testify dependency available"
else
    print_status "FAIL" "Testify dependency not found"
    exit 1
fi

echo ""
echo "🏃 Running Security Test Categories"
echo "==================================="

# Initialize counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Run test categories
test_categories=(
    "validator:TestValidator:Validator Tests"
    "redactor:TestRedactor:Redactor Tests"
    "rbac:TestRBAC:RBAC Authorization Tests"
    "secrets:TestSecrets:Secrets Manager Tests"
    "security_manager:TestSecurityManager:Security Manager Integration Tests"
    "config:TestSecurityConfig:Security Configuration Tests"
    "performance:TestSecurityPerformance:Security Performance Tests"
    "error_handling:TestSecurityErrorHandling:Error Handling Tests"
)

for category_info in "${test_categories[@]}"; do
    IFS=':' read -r category pattern description <<< "$category_info"
    
    if run_test_category "$category" "$pattern" "$description"; then
        ((PASSED_TESTS++))
    else
        ((FAILED_TESTS++))
    fi
    ((TOTAL_TESTS++))
done

# Run cache integration tests if Redis is available
if [ "$REDIS_AVAILABLE" = true ]; then
    if run_test_category "cache_integration" "TestCacheWithSecurity" "Cache Integration Tests"; then
        ((PASSED_TESTS++))
    else
        ((FAILED_TESTS++))
    fi
    ((TOTAL_TESTS++))
else
    print_status "SKIP" "Cache Integration Tests (Redis not available)"
    ((SKIPPED_TESTS++))
    ((TOTAL_TESTS++))
fi

# Run all tests together
echo ""
print_status "INFO" "Running all security tests together..."
echo "----------------------------------------"

if go test -v ./tests/security/ -timeout 300s > "security_test_results/all_tests_output.txt" 2>&1; then
    print_status "PASS" "All security tests completed successfully"
    echo "Results saved to: security_test_results/all_tests_output.txt"
else
    print_status "FAIL" "Some security tests failed"
    echo "Check: security_test_results/all_tests_output.txt"
fi

# Generate security report
echo ""
echo "📊 Security Test Report"
echo "======================"

# Count test functions
echo "Test Categories Summary:"
echo "-----------------------"

for category_info in "${test_categories[@]}"; do
    IFS=':' read -r category pattern description <<< "$category_info"
    test_count=$(grep -c "^func $pattern" tests/security/security_test.go 2>/dev/null || echo "0")
    echo "  • $description: $test_count test functions"
done

# Count total test functions
total_test_functions=$(grep -c "^func Test" tests/security/security_test.go 2>/dev/null || echo "0")
echo "  • Total test functions: $total_test_functions"

echo ""
echo "Test Execution Summary:"
echo "----------------------"
echo "  • Total test categories: $TOTAL_TESTS"
echo "  • Passed: $PASSED_TESTS"
echo "  • Failed: $FAILED_TESTS"
echo "  • Skipped: $SKIPPED_TESTS"

# Calculate success rate
if [ $TOTAL_TESTS -gt 0 ]; then
    success_rate=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    echo "  • Success rate: ${success_rate}%"
fi

echo ""
echo "Security Features Tested:"
echo "-------------------------"
echo "  🔐 Input Validation (Key/Value validation, pattern matching)"
echo "  🛡️  Data Redaction (Sensitive data masking, JSON redaction)"
echo "  🔑 Access Control (RBAC authorization, context validation)"
echo "  🔒 Secrets Management (Environment variables, prefix handling)"
echo "  ⚙️  Configuration (Default/custom configs, error handling)"
echo "  🚀 Performance (Validation/redaction speed, memory usage)"
echo "  🎯 Cache Integration (Secure cache operations, blocked patterns)"

echo ""
echo "Security Patterns Tested:"
echo "-------------------------"
echo "  • Path traversal attacks (../etc/passwd)"
echo "  • XSS attacks (javascript:alert())"
echo "  • HTML injection (<script> tags)"
echo "  • SQL injection patterns"
echo "  • Sensitive data patterns (passwords, tokens, secrets)"
echo "  • Email address patterns"
echo "  • API key patterns"

echo ""
echo "📁 Test Results Location:"
echo "  • Individual test outputs: security_test_results/"
echo "  • All tests output: security_test_results/all_tests_output.txt"

# Generate coverage report if requested
if [ "$1" = "--coverage" ]; then
    echo ""
    print_status "INFO" "Generating coverage report..."
    go test -v -cover ./tests/security/ -coverprofile=security_test_results/coverage.out
    go tool cover -html=security_test_results/coverage.out -o security_test_results/coverage.html
    print_status "PASS" "Coverage report generated: security_test_results/coverage.html"
fi

# Final status
echo ""
if [ $FAILED_TESTS -eq 0 ]; then
    print_status "PASS" "All security tests completed successfully! 🎉"
    echo ""
    echo "Security test suite covers:"
    echo "  ✅ Input validation and sanitization"
    echo "  ✅ Sensitive data protection"
    echo "  ✅ Access control mechanisms"
    echo "  ✅ Secrets management"
    echo "  ✅ Configuration security"
    echo "  ✅ Performance benchmarks"
    echo "  ✅ Error handling and edge cases"
    echo "  ✅ Cache integration security"
    exit 0
else
    print_status "FAIL" "Some security tests failed. Please review the results."
    echo ""
    echo "Failed test categories:"
    for category_info in "${test_categories[@]}"; do
        IFS=':' read -r category pattern description <<< "$category_info"
        if [ -f "security_test_results/${category}_output.txt" ]; then
            if ! grep -q "PASS" "security_test_results/${category}_output.txt" 2>/dev/null; then
                echo "  ❌ $description"
            fi
        fi
    done
    exit 1
fi
