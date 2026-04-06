#!/bin/bash
set -e

echo "Running go mod tidy..."
go mod tidy

echo "Running code generation..."
make gen

echo "Running linter..."
golangci-lint run

echo "Running tests..."
go test ./...

echo "Building project..."
go build ./...

echo "All checks passed!"
