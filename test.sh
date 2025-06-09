#!/bin/bash

# GoTime Test Runner Script
# Runs all tests for the GoTime secret sharing application

echo "🧪 Running GoTime Tests"
echo "======================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    case $1 in
        "PASS") echo -e "${GREEN}✅ $2${NC}" ;;
        "FAIL") echo -e "${RED}❌ $2${NC}" ;;
        "INFO") echo -e "${YELLOW}ℹ️  $2${NC}" ;;
    esac
}

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_status "FAIL" "Go is not installed or not in PATH"
    exit 1
fi

print_status "INFO" "Go version: $(go version)"
echo

# Run go mod tidy to ensure dependencies are up to date
print_status "INFO" "Ensuring dependencies are up to date..."
go mod tidy
if [ $? -ne 0 ]; then
    print_status "FAIL" "Failed to update dependencies"
    exit 1
fi

echo

# Run unit tests
print_status "INFO" "Running unit tests..."
go test -v ./... -run "Test.*Store|TestGenerateID"
if [ $? -eq 0 ]; then
    print_status "PASS" "Unit tests completed successfully"
else
    print_status "FAIL" "Unit tests failed"
    exit 1
fi

echo

# Run handler tests
print_status "INFO" "Running handler tests..."
go test -v ./... -run "Test.*Handler"
if [ $? -eq 0 ]; then
    print_status "PASS" "Handler tests completed successfully"
else
    print_status "FAIL" "Handler tests failed"
    exit 1
fi

echo

# Run integration tests
print_status "INFO" "Running integration tests..."
go test -v ./... -run "TestFull|TestDirect|TestHome|TestView|TestConcurrent"
if [ $? -eq 0 ]; then
    print_status "PASS" "Integration tests completed successfully"
else
    print_status "FAIL" "Integration tests failed"
    exit 1
fi

echo

# Run all tests with coverage
print_status "INFO" "Running full test suite with coverage..."
go test -v -cover ./...
if [ $? -eq 0 ]; then
    print_status "PASS" "All tests completed successfully with coverage report"
else
    print_status "FAIL" "Some tests failed"
    exit 1
fi

echo

# Run race condition tests
print_status "INFO" "Running race condition tests..."
go test -race -v ./... -run "TestConcurrent"
if [ $? -eq 0 ]; then
    print_status "PASS" "Race condition tests completed successfully"
else
    print_status "FAIL" "Race condition tests failed"
    exit 1
fi

echo
print_status "PASS" "🎉 All tests passed! GoTime is ready for deployment."
echo

# Optional: Run benchmarks if they exist
if go test -list=Benchmark ./... | grep -q Benchmark; then
    print_status "INFO" "Running benchmarks..."
    go test -bench=. ./...
fi