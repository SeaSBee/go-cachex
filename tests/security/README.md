# go-cachex Security Tests

This directory contains comprehensive security tests for all security components of the `go-cachex` library.

## Overview

The security tests cover the following security features:

### 🔐 Security Components Tested

1. **Validator** - Input validation for keys and values
2. **Redactor** - Sensitive data redaction from logs and output
3. **RBAC Authorizer** - Role-based access control
4. **Secrets Manager** - Environment-based secrets management
5. **Security Manager** - Integrated security management

### 🛡️ Security Features Covered

#### **Input Validation**
- **Key Validation**: Length limits, pattern matching, blocked patterns
- **Value Validation**: Size limits, nil checks
- **Pattern Matching**: Allowed patterns, blocked patterns (regex-based)
- **Security Patterns**: Path traversal, XSS, HTML injection prevention

#### **Data Redaction**
- **Sensitive Field Redaction**: Passwords, tokens, secrets, API keys
- **Pattern-Based Redaction**: Configurable regex patterns
- **JSON Redaction**: Structured data redaction
- **Custom Redaction**: User-defined patterns and replacements

#### **Access Control**
- **RBAC Authorization**: Role-based access control
- **Context-Based Authorization**: User, role, resource, action validation
- **Authorization Results**: Detailed authorization decisions

#### **Secrets Management**
- **Environment Variables**: Secure secret storage
- **Prefix-Based Secrets**: Configurable secret prefixes
- **Secret Retrieval**: Get and get-with-default operations
- **Secret Listing**: Discovery of available secrets

## Test Categories

### 1. Validator Tests (`TestValidator*`)
- **Key Validation**: Tests for valid/invalid keys, length limits, pattern matching
- **Value Validation**: Tests for valid/invalid values, size limits
- **Security Patterns**: Tests for blocked patterns (path traversal, XSS, etc.)
- **Allowed Patterns**: Tests for whitelist-based validation

### 2. Redactor Tests (`TestRedactor*`)
- **Basic Redaction**: Simple field redaction (password, token, secret)
- **Complex Patterns**: Advanced regex patterns for various data formats
- **JSON Redaction**: Structured data redaction
- **Invalid Patterns**: Error handling for malformed regex patterns

### 3. RBAC Authorization Tests (`TestRBAC*`)
- **Authorization Context**: User, role, resource, action validation
- **Authorization Results**: Decision making and reasoning
- **Default Behavior**: No-op implementation testing

### 4. Secrets Manager Tests (`TestSecrets*`)
- **Secret Retrieval**: Getting secrets from environment variables
- **Default Values**: Fallback behavior for missing secrets
- **Secret Listing**: Discovering available secrets
- **Prefix Management**: Custom prefix handling

### 5. Security Manager Integration Tests (`TestSecurityManager*`)
- **Integrated Validation**: Combined key and value validation
- **Integrated Redaction**: Combined data redaction
- **Integrated Authorization**: Combined access control
- **Integrated Secrets**: Combined secrets management

### 6. Configuration Tests (`TestSecurityConfig*`)
- **Default Configuration**: Testing default security settings
- **Custom Configuration**: Testing custom security settings
- **Configuration Validation**: Error handling for invalid configs

### 7. Cache Integration Tests (`TestCacheWithSecurity*`)
- **Secure Cache Operations**: Cache operations with security validation
- **Blocked Key Patterns**: Prevention of malicious key patterns
- **Data Redaction**: Redaction of sensitive cached data

### 8. Performance Tests (`TestSecurityPerformance*`)
- **Validation Performance**: Speed of key and value validation
- **Redaction Performance**: Speed of data redaction
- **Performance Benchmarks**: Ensuring security doesn't impact performance

### 9. Error Handling Tests (`TestSecurityErrorHandling*`)
- **Invalid Configurations**: Error handling for malformed configs
- **Invalid Patterns**: Error handling for invalid regex patterns
- **Nil Configurations**: Default behavior for nil configs

## Security Patterns Tested

### **Blocked Patterns**
- `^\.\./` - Path traversal attacks
- `[<>\"'&]` - HTML/XML injection
- `javascript:` - XSS attacks
- `data:text/html` - XSS attacks
- `^admin:` - Admin access prevention

### **Redaction Patterns**
- `"password"\s*:\s*"[^"]*"` - JSON password fields
- `"token"\s*:\s*"[^"]*"` - JSON token fields
- `"secret"\s*:\s*"[^"]*"` - JSON secret fields
- `password\s*=\s*[^\s]+` - Key-value password fields
- `api_key\s*:\s*[a-zA-Z0-9]+` - API key fields
- `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b` - Email addresses

### **Allowed Patterns**
- `^user:\d+$` - User ID patterns
- `^product:\d+$` - Product ID patterns
- `^order:\d+$` - Order ID patterns

## Prerequisites

1. **Go Environment**: Go 1.24.5 or later
2. **Redis Server**: For cache integration tests (optional)
3. **Test Dependencies**: `github.com/stretchr/testify`

## Running Security Tests

### Quick Run
```bash
# Run all security tests
go test -v ./tests/security/

# Run specific test category
go test -v -run TestValidator ./tests/security/
go test -v -run TestRedactor ./tests/security/
go test -v -run TestRBAC ./tests/security/
go test -v -run TestSecrets ./tests/security/

# Run with coverage
go test -v -cover ./tests/security/
```

### Test Categories
```bash
# Validator tests
go test -v -run TestValidator ./tests/security/

# Redactor tests
go test -v -run TestRedactor ./tests/security/

# Authorization tests
go test -v -run TestRBAC ./tests/security/

# Secrets management tests
go test -v -run TestSecrets ./tests/security/

# Integration tests
go test -v -run TestSecurityManager ./tests/security/

# Configuration tests
go test -v -run TestSecurityConfig ./tests/security/

# Cache integration tests
go test -v -run TestCacheWithSecurity ./tests/security/

# Performance tests
go test -v -run TestSecurityPerformance ./tests/security/

# Error handling tests
go test -v -run TestSecurityErrorHandling ./tests/security/
```

## Test Data

### **Test Users**
```go
type User struct {
    ID       int    `json:"id"`
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"password"`
    Token    string `json:"token"`
    Secret   string `json:"secret"`
}
```

### **Test Scenarios**
- **Valid Data**: Normal, expected data patterns
- **Malicious Data**: Attack patterns (XSS, path traversal, etc.)
- **Edge Cases**: Empty data, null values, extreme sizes
- **Performance Data**: Large datasets for performance testing

## Security Best Practices Tested

### **Input Validation**
- ✅ Length limits on keys and values
- ✅ Pattern-based validation (whitelist/blacklist)
- ✅ Security pattern blocking (XSS, path traversal, etc.)
- ✅ Null/empty value handling

### **Data Protection**
- ✅ Sensitive data redaction
- ✅ Configurable redaction patterns
- ✅ JSON field redaction
- ✅ Log data protection

### **Access Control**
- ✅ Role-based authorization
- ✅ Context-based decisions
- ✅ Authorization result tracking
- ✅ Default security policies

### **Secrets Management**
- ✅ Environment-based secrets
- ✅ Prefix-based organization
- ✅ Default value fallbacks
- ✅ Secret discovery

### **Error Handling**
- ✅ Graceful error handling
- ✅ Detailed error messages
- ✅ Configuration validation
- ✅ Fallback behaviors

## Performance Considerations

### **Validation Performance**
- Key validation: < 1ms per operation
- Value validation: < 1ms per operation
- Pattern matching: Optimized regex compilation

### **Redaction Performance**
- Text redaction: < 1ms per operation
- JSON redaction: < 1ms per operation
- Pattern compilation: Cached compiled patterns

### **Memory Usage**
- Minimal memory overhead
- Efficient pattern storage
- Garbage collection friendly

## Security Recommendations

### **Production Usage**
1. **Configure Blocked Patterns**: Always configure blocked patterns for your use case
2. **Use Allowed Patterns**: Implement whitelist-based validation where possible
3. **Enable Redaction**: Always enable redaction for sensitive data
4. **Secure Secrets**: Use proper secrets management in production
5. **Monitor Performance**: Monitor security component performance

### **Configuration Best Practices**
1. **Custom Validation Rules**: Define custom validation rules for your domain
2. **Comprehensive Redaction**: Include all sensitive field patterns
3. **Proper Authorization**: Implement proper RBAC for your application
4. **Secret Organization**: Use consistent secret naming and prefixes

## Troubleshooting

### **Common Issues**

1. **Validation Failures**
   ```
   Error: key matches blocked pattern
   ```
   **Solution**: Review blocked patterns and adjust if needed

2. **Redaction Not Working**
   ```
   Sensitive data still visible in logs
   ```
   **Solution**: Check redaction patterns and ensure they match your data format

3. **Performance Issues**
   ```
   Security operations taking too long
   ```
   **Solution**: Review pattern complexity and optimize regex patterns

4. **Secret Not Found**
   ```
   Error: secret not found
   ```
   **Solution**: Check environment variable names and prefixes

### **Debugging Tips**

1. **Enable Verbose Logging**: Use `-v` flag for detailed test output
2. **Check Pattern Matching**: Verify regex patterns with online testers
3. **Monitor Performance**: Use performance tests to identify bottlenecks
4. **Review Configuration**: Ensure security config matches your requirements

## Contributing

When adding new security tests:

1. **Follow Naming Convention**: Use `Test<Component><Feature>` naming
2. **Include Edge Cases**: Test both valid and invalid scenarios
3. **Add Performance Tests**: Include performance benchmarks for new features
4. **Document Patterns**: Document any new security patterns
5. **Update README**: Update this README with new test categories

## Notes

- All tests use proper cleanup to prevent resource leaks
- Environment variables are restored after tests
- Redis integration tests are skipped if Redis is not available
- Performance tests ensure security doesn't impact application performance
- Error handling tests verify graceful failure modes
