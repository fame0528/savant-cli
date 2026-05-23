---
name: go-testing
description: Guide for writing and running Go tests, benchmarks, and fuzz tests.
license: MIT
compatibility: savant-cli>=0.1.0
---

# Go Testing Guide

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests in a specific package
go test ./internal/agent/

# Run a specific test
go test ./internal/agent/ -run TestAgentRun

# Verbose output
go test -v ./...

# Run with race detector
go test -race ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Writing Tests

Test files are named `*_test.go` in the same package:

```go
package agent

import "testing"

func TestAgentRun(t *testing.T) {
    // Arrange
    agent := NewTestAgent()
    
    // Act
    result, err := agent.Run(context.Background(), "test input")
    
    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != expected {
        t.Errorf("got %q, want %q", result, expected)
    }
}
```

## Table-Driven Tests

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive", 1, 2, 3},
        {"negative", -1, -2, -3},
        {"zero", 0, 0, 0},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Add(tt.a, tt.b)
            if got != tt.expected {
                t.Errorf("Add(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.expected)
            }
        })
    }
}
```

## Benchmarks

```go
func BenchmarkAgentRun(b *testing.B) {
    for i := 0; i < b.N; i++ {
        agent.Run(context.Background(), "test")
    }
}
```

Run with: `go test -bench=. ./...`

## Key Patterns

- Use `t.Helper()` in test helper functions
- Use `t.Cleanup()` for teardown
- Use `t.TempDir()` for temporary directories
- Use `t.Parallel()` for independent tests
- Never ignore errors in tests
