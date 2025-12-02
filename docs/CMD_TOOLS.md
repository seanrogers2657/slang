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

# Build with verbose debug output
go run cmd/sl/main.go build --verbose example.sl

# Use native assembler instead of system assembler
go run cmd/sl/main.go build --assembler native example.sl
```

**Commands:**
- `build <file>` - Compile a .sl file to an executable
- `run <file>` - Compile and execute a .sl file

**Flags:**
- `--assembler, -a` - Assembler backend: "system" (default) or "native"
- `--verbose, -v` - Enable verbose debug output for all compilation stages

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

### slasm - Standalone ARM64 Assembler
**Location:** `cmd/slasm/`

Standalone assembler/linker for ARM64 assembly files. Can use either the system assembler (`as`/`ld`) or the native slasm implementation.

```bash
# Assemble an assembly file to object file
go run cmd/slasm/main.go assemble -o output.o input.s

# Link object files to executable
go run cmd/slasm/main.go link -o output output.o

# Assemble and link in one step
go run cmd/slasm/main.go build -o output input.s

# Use verbose output
go run cmd/slasm/main.go build -o output --verbose input.s
```

**Commands:**
- `assemble, a` - Assemble an ARM64 assembly file to object file
- `link, l` - Link object files to create an executable
- `build, b` - Assemble and link in one step

**Common Flags:**
- `--output, -o` - Output file path (required)
- `--verbose, -v` - Enable verbose debug output
- `--backend` - Assembler backend: "system" (default) or "native"

**Link-specific Flags:**
- `--arch` - Target architecture (default: arm64)
- `--sdk` - SDK path for linking
- `--entry` - Entry point symbol (default: _start)
- `--no-system` - Don't link against libSystem

**Build-specific Flags:**
- `--keep-intermediates` - Keep intermediate object files

See: [SLASM Documentation](SLASM_README.md)

## Debug Tools

### slasm-debug - Assembler Debug Tool
**Location:** `cmd/slasm-debug/`

Debug tool for inspecting slasm internals (tokens, AST, symbol table, etc.).

See: [Debug Guide](SLASM_DEBUG_GUIDE.md)

## Usage Patterns

### Running Commands

```bash
# Direct execution with go run
go run cmd/sl/main.go build example.sl
go run cmd/slm/main.go test
go run cmd/slasm/main.go build -o output input.s

# Build and install
go build -o sl cmd/sl/main.go
./sl build example.sl

# Install to GOPATH/bin
go install ./cmd/sl
go install ./cmd/slm
go install ./cmd/slasm
```

### Project Workflow

```bash
# 1. Make changes to code
# 2. Run quality checks
go run cmd/slm/main.go check

# 3. Run specific tests
go run cmd/slm/main.go test-parser

# 4. Debug assembler issues (use verbose flag)
go run cmd/slasm/main.go build -o output --verbose input.s

# 5. Test the compiler
go run cmd/sl/main.go run _examples/slang/add.sl

# 6. Run e2e tests
go test ./test/sl/...    # Slang compiler e2e tests
go test ./test/slasm/... # Assembler e2e tests
```

## Documentation

- [Project README](../README.md) - Main project documentation
- [SLASM Documentation](SLASM_README.md) - Assembler documentation
- [Debug Guide](SLASM_DEBUG_GUIDE.md) - Debugging tools
- [Testing Guide](TESTING.md) - Testing documentation
