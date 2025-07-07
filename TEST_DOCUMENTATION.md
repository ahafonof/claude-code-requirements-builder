# Testing Documentation

## Overview

This document provides comprehensive guidance for the testing infrastructure of the Claude Code Requirements Builder project. The test suite achieves **88.6% code coverage** through a combination of unit tests, integration tests, and careful test design that validates both happy paths and error scenarios.

## Test Suite Structure

The project follows Go's standard testing conventions with test files colocated alongside source files:

```
.
├── activity_feed.go          # Activity feed implementation
├── activity_feed_test.go     # Activity feed tests
├── distributed_ratelimiter.go    # Distributed rate limiter
├── distributed_ratelimiter_test.go # Rate limiter tests
├── main.go                   # Main application
├── main_test.go              # Main application tests
├── ratelimiter.go            # Local rate limiter
└── ratelimiter_test.go       # Local rate limiter tests
```

### Test Organization by Component

1. **Activity Feed Tests** (`activity_feed_test.go`)
   - Event structure validation
   - Concurrent operations testing
   - SSE broadcasting functionality
   - Event emission patterns

2. **Distributed Rate Limiter Tests** (`distributed_ratelimiter_test.go`)
   - Redis integration scenarios
   - Circuit breaker behavior
   - Fallback mechanisms
   - Metrics collection

3. **Local Rate Limiter Tests** (`ratelimiter_test.go`)
   - Token bucket algorithm
   - Concurrent access patterns
   - IP extraction logic
   - Cleanup mechanisms

4. **Main Application Tests** (`main_test.go`)
   - HTTP endpoint testing
   - Handler validation
   - Response format verification

## Test Coverage Summary

### Overall Coverage: 88.6%

### Coverage by Package Component:

| Component | Coverage | Description |
|-----------|----------|-------------|
| Activity Feed | 100% | Full coverage of event handling and broadcasting |
| Local Rate Limiter | 94.1% | Comprehensive token bucket testing |
| HTTP Handlers | 100% | All endpoints fully tested |
| Distributed Rate Limiter | 80% | Redis integration with fallback scenarios |
| SSE Broadcasting | 100% | Real-time event streaming validation |
| Circuit Breaker | 80% | State transitions and recovery testing |

### Coverage by File:

```
activity_feed.go: 100%
- NewActivityFeed: 100%
- AddEvent: 100%
- GetRecentEvents: 100%
- SSE Broadcasting: 100%
- Event Emission: 100%

ratelimiter.go: 94.1%
- initializeRateLimiter: 94.1%
- RateLimitMiddleware: 92.9%
- Token bucket operations: 100%
- IP extraction: 100%
- Cleanup routines: 100%

distributed_ratelimiter.go: 80%
- NewDistributedRateLimiter: 100%
- Allow/AllowWithRequest: 80%
- Circuit breaker logic: 80%
- Redis operations: 77.8%
- Metrics collection: 87.5%

main.go: 71.4%
- HTTP handlers: 100%
- SSE handler: 71.4%
- main() function: 0% (entry point)
```

## Running Tests

### Basic Test Execution

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific test file
go test -v activity_feed_test.go activity_feed.go

# Run specific test function
go test -v -run TestActivityFeed
```

### Coverage Analysis

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# View coverage by function (sorted)
go tool cover -func=coverage.out | sort -k3 -n
```

### Continuous Testing

```bash
# Watch mode (requires external tool)
# Install: go install github.com/smartystreets/goconvey
goconvey

# Or use a simple watch script
while true; do
    clear
    go test -cover ./...
    sleep 2
done
```

## Key Test Scenarios

### 1. Rate Limiting Tests

**Token Bucket Algorithm**
- Verifies requests are allowed up to the limit
- Confirms requests are blocked when limit exceeded
- Tests token replenishment over time
- Validates concurrent access safety

**Distributed Rate Limiting**
- Redis-based rate limiting with Lua scripts
- Fallback to local limiting on Redis failure
- Circuit breaker activation on repeated failures
- Metrics tracking for monitoring

### 2. Activity Feed Tests

**Event Management**
- Event creation and storage
- Recent events retrieval with proper ordering
- Concurrent event addition
- Memory management for event buffer

**SSE Broadcasting**
- Client subscription/unsubscription
- Concurrent broadcaster operations
- Event serialization and transmission
- Connection management

### 3. HTTP Endpoint Tests

**API Endpoints**
- Health check endpoint validation
- Rate-limited endpoints (users, products)
- Proper HTTP status codes
- JSON response formatting

**SSE Endpoint**
- Server-Sent Events setup
- Proper headers and content type
- Event stream formatting

### 4. Error Handling Tests

**Graceful Degradation**
- Redis connection failures
- Circuit breaker state transitions
- Fallback mechanism activation
- Error recovery patterns

**Edge Cases**
- Empty IP addresses
- Malformed requests
- Concurrent modifications
- Resource cleanup

## Test Patterns and Best Practices

### Table-Driven Tests

The codebase extensively uses table-driven tests for comprehensive scenario coverage:

```go
tests := []struct {
    name           string
    method         string
    expectedStatus int
    expectedBody   Response
}{
    {
        name:           "successful_health_check",
        method:         http.MethodGet,
        expectedStatus: http.StatusOK,
        expectedBody: Response{
            Message: "API is healthy",
            Status:  http.StatusOK,
        },
    },
    // More test cases...
}
```

### Helper Functions

- `testConfig()` - Creates consistent test configurations
- `skipIfRedisUnavailable()` - Skips Redis tests when unavailable
- `resetRateLimiter()` - Ensures clean state between tests

### Concurrent Testing

Tests validate thread-safety through concurrent operations:

```go
// Concurrent event addition test
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        feed.AddEvent(event)
    }(i)
}
wg.Wait()
```

## Known Limitations

### 1. Redis Integration Tests
- Require Redis server running locally
- Tests skip automatically if Redis unavailable
- May need Docker setup for CI environments

### 2. Time-Dependent Tests
- Rate limiter tests depend on timing
- May be flaky on heavily loaded systems
- Consider using time mocking for deterministic tests

### 3. Coverage Gaps
- `main()` function (0% - typical for entry points)
- Some error paths in recovery monitoring (55.6%)
- Redis failure scenarios need expanded coverage

## Future Improvements

### 1. Enhanced Test Coverage
- **Target: 95% coverage**
- Add more Redis failure scenarios
- Test recovery monitoring edge cases
- Add integration tests for full request flow

### 2. Performance Testing
- Benchmark rate limiting algorithms
- Load testing for concurrent requests
- Memory usage profiling
- Latency measurements

### 3. Test Infrastructure
- Docker Compose for test dependencies
- Automated test data generation
- Property-based testing for edge cases
- Mutation testing for test quality

### 4. Additional Test Types
- End-to-end integration tests
- Chaos engineering tests
- Security testing (rate limit bypass attempts)
- Compatibility testing across Go versions

## CI/CD Integration

### GitHub Actions Configuration

```yaml
name: Tests

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Install dependencies
      run: go mod download
    
    - name: Run tests
      run: go test -v -cover -coverprofile=coverage.out ./...
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
        flags: unittests
        name: codecov-umbrella
```

### Local Pre-commit Hook

```bash
#!/bin/sh
# .git/hooks/pre-commit

# Run tests before commit
go test -cover ./... || {
    echo "Tests failed. Commit aborted."
    exit 1
}

# Check coverage threshold
coverage=$(go test -cover ./... | grep -oE '[0-9]+\.[0-9]+%' | sed 's/%//')
threshold=85.0

if (( $(echo "$coverage < $threshold" | bc -l) )); then
    echo "Coverage $coverage% is below threshold $threshold%"
    exit 1
fi

echo "All tests passed with $coverage% coverage"
```

## Maintenance Guidelines

### Adding New Tests

1. **Follow Naming Conventions**
   - Test functions: `Test{Function}_{Scenario}_{ExpectedResult}`
   - Helper functions: `test{Purpose}` or `setup{Component}`

2. **Maintain Test Independence**
   - Each test should be runnable in isolation
   - Use `t.Run()` for subtests
   - Clean up resources in defer statements

3. **Document Complex Tests**
   - Add comments explaining test purpose
   - Document any special setup requirements
   - Explain expected behaviors

### Reviewing Test Coverage

1. **Regular Coverage Audits**
   - Run coverage reports weekly
   - Identify untested code paths
   - Prioritize critical path coverage

2. **Coverage Goals**
   - Maintain minimum 85% coverage
   - 100% coverage for critical components
   - Document justified coverage exclusions

3. **Test Quality Metrics**
   - Monitor test execution time
   - Track flaky test occurrences
   - Measure test maintenance burden

## Troubleshooting

### Common Issues

1. **Redis Connection Failures**
   ```bash
   # Start Redis locally
   docker run -d -p 6379:6379 redis:7-alpine
   
   # Or install via brew (macOS)
   brew services start redis
   ```

2. **Timing-Related Failures**
   - Increase timeouts in CI environments
   - Use time.Sleep() sparingly
   - Consider time mocking libraries

3. **Port Conflicts**
   - Test servers use random ports
   - Check for port availability
   - Use httptest.NewServer() for isolation

### Debug Techniques

```bash
# Run specific test with detailed output
go test -v -run TestDistributedRateLimiter/with_Redis

# Debug with delve
dlv test -- -test.run TestActivityFeed

# Race condition detection
go test -race ./...

# CPU profiling
go test -cpuprofile=cpu.prof -bench=.
```

## Conclusion

The test suite provides comprehensive validation of the Claude Code Requirements Builder's functionality. With 88.6% coverage and well-structured test scenarios, the codebase maintains high quality and reliability. Continuous improvements in test coverage, performance testing, and CI/CD integration will further enhance the project's robustness and maintainability.