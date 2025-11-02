# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Slang** is a compiler for a simple programming language written in Go. It targets ARM64 assembly for macOS. The compiler follows a traditional three-stage pipeline:

1. **Lexer** (`frontend/lexer`) - Tokenizes source code into tokens
2. **Parser** (`frontend/parser`) - Builds an Abstract Syntax Tree (AST) from tokens
3. **Code Generator** (`backend/as`) - Generates ARM64 assembly from the AST

The compiler currently supports binary expressions with integers and the following operators:
- Arithmetic: `+`, `-`, `*`, `/`, `%`
- Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=`

## Development Commands

### Building and Running

```bash
# Build the compiler binary
make build

# Run compiler on the example file
make run

# Compile, assemble, link, and execute
make run-and-test
```

### Testing

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Generate HTML coverage report
make test-coverage

# Run specific component tests
make test-lexer        # Frontend lexer only
make test-parser       # Frontend parser only
make test-codegen      # Backend assembly generator only
make test-integration  # End-to-end integration tests

# Run single test by name
go test -run TestLexerNumbers
go test -run TestEndToEndCompilation
```

### Code Quality

```bash
# Format code
make fmt

# Run linter (go vet)
make lint

# Run all quality checks (fmt + lint + test)
make check

# Clean build artifacts
make clean
```

## Architecture

### Compilation Pipeline

The pipeline is orchestrated in `main.go`:

```
Source Code (.sl file)
    ↓
Lexer (tokenization)
    ↓
Parser (AST construction)
    ↓
Code Generator (ARM64 assembly)
    ↓
Assembly Output (.s file)
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
- Outputs: `*Expr` and `[]error`

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
- **SDK Path**: Hardcoded in `main.go:59` to macOS 15.5 SDK
- **Assembler**: Uses macOS `as` command
- **Linker**: Uses macOS `ld` with `-lSystem` for system calls

## Testing Strategy

The project has comprehensive test coverage (74.3% overall):
- Backend (Assembly Generator): 100%
- Frontend (Lexer): 96.7%
- Frontend (Parser): 93.8%

All tests follow table-driven patterns with subtests using `t.Run()`. See `TESTING.md` for detailed testing documentation.

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
