# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Slang** is a compiler for a simple programming language written in Go. It targets ARM64 assembly for macOS. The compiler follows a traditional four-stage pipeline:

1. **Lexer** (`frontend/lexer`) - Tokenizes source code into tokens
2. **Parser** (`frontend/parser`) - Builds an Abstract Syntax Tree (AST) from tokens
3. **Semantic Analyzer** (`frontend/semantic`) - Performs type checking and semantic analysis
4. **Code Generator** (`backend/as`) - Generates ARM64 assembly from the AST

The compiler currently supports binary expressions with integers and the following operators:
- Arithmetic: `+`, `-`, `*`, `/`, `%`
- Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=`

## Development Commands

The project uses a custom Go-based build tool called `slm` (slang make) located at `cmd/slm/main.go` instead of a traditional Makefile. This provides a cross-platform, type-safe build system using the `urfave/cli/v2` framework.

### Running the Build Tool

You can run the build tool in several ways:

```bash
# Directly with go run (recommended for development)
go run cmd/slm/main.go <command>

# Build a binary first
go build -o slm cmd/slm/main.go
./slm <command>

# Install globally
go install ./cmd/slm
slm <command>  # If $GOPATH/bin is in your PATH
```

### Building and Running

The `sl` compiler has two main commands:

**`build`** - Compiles a Slang source file to an executable:
```bash
# Build the sl compiler
go build -o sl cmd/sl/main.go

# Compile a .sl source file
./sl build <source-file>
go run cmd/sl/main.go build <source-file>
```

**`run`** - Compiles and executes a Slang source file in one step:
```bash
# Compile and run a .sl source file
./sl run <source-file>
go run cmd/sl/main.go run <source-file>

# Examples
./sl run _examples/slang/add.sl
./sl run _examples/slang/subtract.sl
```

The build tool (`slm`) also provides convenience commands:
```bash
# Build the compiler binary
go run cmd/slm/main.go build

# Run compiler on the example file
go run cmd/slm/main.go run

# Compile, assemble, link, and execute
go run cmd/slm/main.go run-and-test
```

### Testing

```bash
# Run all tests
go run cmd/slm/main.go test

# Run tests with verbose output
go run cmd/slm/main.go test-verbose

# Generate HTML coverage report
go run cmd/slm/main.go test-coverage

# Run specific component tests
go run cmd/slm/main.go test-lexer        # Frontend lexer only
go run cmd/slm/main.go test-parser       # Frontend parser only
go run cmd/slm/main.go test-codegen      # Backend assembly generator only
go run cmd/slm/main.go test-integration  # End-to-end integration tests

# Additional test commands
go run cmd/slm/main.go test-report       # Detailed pass/fail report
go run cmd/slm/main.go test-race         # Run with race detector
go run cmd/slm/main.go bench             # Run benchmarks

# Run single test by name (use go test directly)
go test -run TestLexerNumbers
go test -run TestEndToEndCompilation
```

### Code Quality

```bash
# Format code
go run cmd/slm/main.go fmt

# Run linter (go vet)
go run cmd/slm/main.go lint

# Run all quality checks (fmt + lint + test)
go run cmd/slm/main.go check

# Clean build artifacts
go run cmd/slm/main.go clean

# Install/update dependencies
go run cmd/slm/main.go deps
```

## Architecture

### Compilation Pipeline

The pipeline is orchestrated in `cmd/sl/main.go`. The compiler provides two commands:
- **`build`** - Compiles source to executable (assembly, object, and binary files)
- **`run`** - Compiles and executes in one step

```
Source Code (.sl file)
    ↓
Lexer (tokenization)
    ↓
Parser (AST construction)
    ↓
Semantic Analyzer (type checking & validation)
    ↓
Code Generator (ARM64 assembly)
    ↓
Assembly Output (.s file)
    ↓
Assembler (as) → Object file (.o)
    ↓
Linker (ld) → Executable binary
    ↓
[Optional: Execute] (run command only)
```

### Key Data Structures

**Lexer** (`frontend/lexer/lexer.go`):
- `Token` - Represents a single token with `Type` and `Value`
- `TokenType` - Enum for token types (Integer, Plus, Minus, etc.)
- Outputs: `[]Token` and `[]error`

**Parser** (`frontend/parser/parser.go`):
- `Literal` - Represents number literals with type and value
- `Expr` - Binary expression with `Left`, `Op`, and `Right` fields
- Currently only supports binary expressions (no operator precedence or parentheses)
- Outputs: `*Program` and `[]error`

**Semantic Analyzer** (`frontend/semantic/analyzer.go`):
- `Analyzer` - Performs type checking and semantic validation
- `Type` interface - Represents types in the type system (IntegerType, StringType, etc.)
- `TypedProgram` - AST annotated with type information
- Type checking rules:
  - Arithmetic operators (`+`, `-`, `*`, `/`, `%`) require integer operands
  - Comparison operators (`==`, `!=`, `<`, `>`, `<=`, `>=`) require integer operands
  - Print statements can handle any type
- Outputs: `[]*CompilerError` and `*TypedProgram`

**Error Framework** (`frontend/errors/`):
- `CompilerError` - Rich error type with filename, position, and error span
- `FormatError()` - Formats errors with source code context and color highlighting
- Beautiful error messages showing:
  - Error type and message
  - File location (file:line:column)
  - Source code snippet
  - Visual error pointer (^ under the error)
  - Optional hints for fixing the error
  - Summary of total errors/warnings

**Code Generator** (`backend/as/as.go`):
- `AsGenerator` interface with `Generate() (string, error)`
- Generates ARM64 assembly targeting macOS
- All operations store result in register `x2`
- Uses ARM64 instructions: `add`, `sub`, `mul`, `sdiv`, `cmp`, `cset`, `msub`

### ARM64 Assembly Details

Generated assembly follows this structure:
```asm
.global _start
.align 4
_start:
    mov x0, #<left_operand>
    mov x1, #<right_operand>
    <operation>              # Result stored in x2
    mov x0, #1
    mov x16, #0
    svc #0
```

**Arithmetic operations**:
- Addition: `add x2, x0, x1`
- Subtraction: `sub x2, x0, x1`
- Multiplication: `mul x2, x0, x1`
- Division: `sdiv x2, x0, x1`
- Modulo: `sdiv x3, x0, x1` + `msub x2, x3, x1, x0`

**Comparison operations** (result is 0 or 1 in x2):
- Equal: `cmp x0, x1` + `cset x2, eq`
- Not Equal: `cmp x0, x1` + `cset x2, ne`
- Less Than: `cmp x0, x1` + `cset x2, lt`
- Greater Than: `cmp x0, x1` + `cset x2, gt`
- Less/Equal: `cmp x0, x1` + `cset x2, le`
- Greater/Equal: `cmp x0, x1` + `cset x2, ge`

### Platform Requirements

- **Target**: ARM64 macOS only
- **SDK Path**: Hardcoded in `cmd/sl/main.go` (build and run commands) to macOS 15.5 SDK
- **Assembler**: Uses macOS `as` command
- **Linker**: Uses macOS `ld` with `-lSystem` for system calls
- **Output Files**:
  - `build/output.s` - Generated assembly code
  - `build/output.o` - Assembled object file
  - `build/output` - Linked executable (run command)

## Testing Strategy

The project has comprehensive test coverage:
- Backend (Assembly Generator): 62.5%
- Frontend (Lexer): 96.7%
- Frontend (Parser): 60.4%
- Frontend (Semantic Analyzer): 62.2%
- Frontend (Error Framework): 89.6%

All tests follow table-driven patterns with subtests using `t.Run()`. See `TESTING.md` for detailed testing documentation.

**Semantic Analysis Tests** (`frontend/semantic/analyzer_test.go`):
- Type checking for all operators (arithmetic and comparison)
- Type error detection and reporting
- Multi-statement program analysis
- Integration with error framework

### Integration Tests

`integration_test.go` contains end-to-end tests that verify the entire pipeline:
- `TestEndToEndCompilation` - Full source-to-assembly compilation
- `TestCompilationPipelineStages` - Individual stage verification
- `TestExampleFile` - Example file validation
- `TestRegressions` - Edge cases (newlines, no whitespace, large numbers)

## Adding New Features

### Adding a New Operator

When adding a new operator, you must update three files:

1. **Lexer** (`frontend/lexer/lexer.go`):
   - Add `TokenType<OperatorName>` constant to the enum
   - Add parsing logic in the `Parse()` method

2. **Parser** (`frontend/parser/parser.go`):
   - Add case in `ParseBinaryExpression()` switch statement

3. **Code Generator** (`backend/as/as.go`):
   - Add case in `GenerateExpr()` switch statement with ARM64 instructions

4. **Tests**: Update test files for all three components

### Current Limitations

- Only supports binary expressions (no complex expressions or operator precedence)
- No parentheses support
- No variables, functions, or control flow
- Hardcoded exit syscall in generated assembly
- Assembly is not executed in tests (only structure is verified)

## Module Information

- **Module Path**: `github.com/seanrogers2657/slang`
- **Go Version**: 1.24
- **Dependencies**:
  - `github.com/davecgh/go-spew` - Debug printing
  - `github.com/urfave/cli/v2` - CLI framework
