# Command-Line Tools

This directory contains all executable commands for the Slang project.

## Compiler Tools

### sl - Slang Compiler
**Location:** `cmd/sl/`

The main Slang compiler. Compiles `.sl` source files to executable binaries.

```bash
# Build a Slang program
go run cmd/sl/main.go build example.sl

# Compile and run
go run cmd/sl/main.go run example.sl
```

**Commands:**
- `build <file>` - Compile a .sl file to an executable
- `run <file>` - Compile and execute a .sl file

See: [Main project README](../README.md)

## Build Tools

### slm - Slang Make
**Location:** `cmd/slm/`

Build tool for the Slang project (alternative to Makefiles).

```bash
# Build the compiler
go run cmd/slm/main.go build

# Run tests
go run cmd/slm/main.go test

# Run with coverage
go run cmd/slm/main.go test-coverage
```

**Available commands:**
- `build` - Build the sl compiler
- `run` - Run compiler on example file
- `test` - Run all tests
- `test-verbose` - Run tests with verbose output
- `test-coverage` - Generate HTML coverage report
- `test-lexer`, `test-parser`, `test-codegen`, `test-integration` - Component tests
- `fmt` - Format code
- `lint` - Run linter
- `check` - Run fmt + lint + test
- `clean` - Clean build artifacts

## Assembler Tools

### slasm-debug - Assembler Debugger
**Location:** `cmd/slasm-debug/`

Diagnostic tool for the slasm assembler. Shows detailed output from every stage of the assembly pipeline.

```bash
# Run debug build
go run cmd/slasm-debug/main.go
```

**Output includes:**
- Lexer tokens
- Parser IR
- Symbol table
- Instruction encoding (with hex bytes)
- Mach-O structure
- Verification steps (otool, codesign)

See: [SLASM Debug Guide](../docs/SLASM_DEBUG_GUIDE.md)

### slasm - Standalone Assembler
**Location:** `cmd/slasm/`

Standalone slasm assembler command (if available).

### slasm-it - Assembler Integration Tests
**Location:** `cmd/slasm-it/`

Integration test runner for the slasm assembler (if available).

## Testing Tools

### it - Integration Tests
**Location:** `cmd/it/`

Integration test runner for the Slang compiler (if available).

## Usage Patterns

### Running Commands

```bash
# Direct execution with go run
go run cmd/sl/main.go build example.sl
go run cmd/slm/main.go test
go run cmd/slasm-debug/main.go

# Build and install
go build -o sl cmd/sl/main.go
./sl build example.sl

# Install to GOPATH/bin
go install ./cmd/sl
go install ./cmd/slm
go install ./cmd/slasm-debug
```

### Project Workflow

```bash
# 1. Make changes to code
# 2. Run quality checks
go run cmd/slm/main.go check

# 3. Run specific tests
go run cmd/slm/main.go test-parser

# 4. Debug assembler issues
go run cmd/slasm-debug/main.go

# 5. Test the compiler
go run cmd/sl/main.go run _examples/slang/add.sl
```

## Documentation

- [Project README](../README.md) - Main project documentation
- [SLASM Documentation](../docs/SLASM_README.md) - Assembler documentation
- [Debug Guide](../docs/SLASM_DEBUG_GUIDE.md) - Debugging tools
