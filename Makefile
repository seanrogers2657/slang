.PHONY: test test-verbose test-coverage test-lexer test-parser test-codegen test-integration clean build run

# Run all tests
test:
	@echo "Running all tests..."
	@go test ./... -v

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	@go test ./... -v -count=1

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run only lexer tests
test-lexer:
	@echo "Running lexer tests..."
	@go test ./frontend/lexer/... -v

# Run only parser tests
test-parser:
	@echo "Running parser tests..."
	@go test ./frontend/parser/... -v

# Run only code generation tests
test-codegen:
	@echo "Running code generation tests..."
	@go test ./backend/as/... -v

# Run only integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -run TestEndToEnd -v
	@go test -run TestCompilationPipeline -v
	@go test -run TestExampleFile -v
	@go test -run TestRegressions -v

# Run tests and report which ones passed/failed
test-report:
	@echo "Running tests with detailed report..."
	@go test ./... -v -json | grep -E '"Test"|"Pass"|"Fail"'

# Run benchmarks (for future use)
bench:
	@echo "Running benchmarks..."
	@go test ./... -bench=. -benchmem

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	@go test ./... -race -v

# Clean test artifacts
clean:
	@echo "Cleaning test artifacts..."
	@rm -f coverage.out coverage.html
	@rm -rf build/*.o build/*.s build/output build/simple
	@echo "Clean complete"

# Build the compiler
build:
	@echo "Building compiler..."
	@go build -o slang-compiler .
	@echo "Build complete: slang-compiler"

# Run the compiler with the example file
run:
	@echo "Running compiler on example file..."
	@go run . _examples/slang/simple.sl

# Run compiler and test the output
run-and-test:
	@echo "Compiling example..."
	@go run . _examples/slang/simple.sl
	@echo "Assembling output..."
	@as build/output.s -o build/simple.o
	@ld build/simple.o -o build/simple -lSystem -syslibroot $(shell xcrun -sdk macosx --show-sdk-path) -e _start -arch arm64
	@echo "Running compiled binary..."
	@./build/simple
	@echo "Exit code: $$?"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies installed"

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Formatting complete"

# Run linter
lint:
	@echo "Running linter..."
	@go vet ./...
	@echo "Linting complete"

# Run all quality checks
check: fmt lint test
	@echo "All quality checks passed!"

# Display help
help:
	@echo "Available targets:"
	@echo "  test              - Run all tests"
	@echo "  test-verbose      - Run tests with verbose output"
	@echo "  test-coverage     - Run tests and generate coverage report"
	@echo "  test-lexer        - Run only lexer tests"
	@echo "  test-parser       - Run only parser tests"
	@echo "  test-codegen      - Run only code generation tests"
	@echo "  test-integration  - Run only integration tests"
	@echo "  test-report       - Run tests with detailed pass/fail report"
	@echo "  test-race         - Run tests with race detector"
	@echo "  bench             - Run benchmarks"
	@echo "  clean             - Remove test artifacts and build files"
	@echo "  build             - Build the compiler binary"
	@echo "  run               - Run compiler on example file"
	@echo "  run-and-test      - Compile, assemble, link, and run example"
	@echo "  deps              - Install dependencies"
	@echo "  fmt               - Format source code"
	@echo "  lint              - Run linter"
	@echo "  check             - Run fmt, lint, and test"
	@echo "  help              - Display this help message"
