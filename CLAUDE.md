# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Organization

This repository follows strict organizational rules:

1. **Documentation**: All documentation files (README files, markdown files, guides, etc.) must be placed in the `docs/` directory. The only exceptions are:
   - `CLAUDE.md` - This file, which must remain in the root for Claude Code to find
   - `README.md` in root (if present) - Project root readme
   - `go.mod` and `go.sum` - Go module files

2. **Command-line Tools**: All main functions and executable tools must be in the `cmd/` directory, with each tool in its own subdirectory:
   - `cmd/sl/` - The Slang compiler
   - `cmd/slm/` - The build tool
   - `cmd/slasm/` - The assembler
   - `cmd/slasm-debug/` - Assembler debug tool
   - etc.

3. **End-to-End Tests**: All e2e tests must be in the `test/` directory with packages for each tool:
   - `test/sl/` - Slang compiler e2e tests (reads from `_examples/slang/*.sl`)
   - `test/slasm/` - Assembler e2e tests (reads from `_examples/arm64/*.s`)
   - `test/testutil/` - Shared test utilities (expectation parsing)

4. **Testing Documentation**: Testing guides and documentation should be in `docs/` (e.g., `docs/TESTING.md`)

## Project Overview

**Slang** is a compiler for a simple programming language written in Go. It targets ARM64 assembly for macOS. The compiler follows a traditional four-stage pipeline:

1. **Lexer** (`frontend/lexer`) - Tokenizes source code into tokens
2. **Parser** (`frontend/parser`) - Builds an Abstract Syntax Tree (AST) from tokens
3. **Semantic Analyzer** (`frontend/semantic`) - Performs type checking and semantic analysis
4. **Code Generator** (`backend/codegen`) - Generates ARM64 assembly from the AST

The compiler currently supports:
- **Variables**: Immutable (`val`) and mutable (`var`) variables (e.g., `val x = 5`, `var y = 10`)
- **Expressions**: Binary expressions with integers
- **Operators**:
  - Arithmetic: `+`, `-`, `*`, `/`, `%`
  - Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=`
- **Statements**: Expression statements, variable declarations
- **Functions**: Function declarations with `fn` keyword (e.g., `fn main() { ... }`)
- **Built-in Functions**:
  - `print(value)` - print an integer value to stdout
  - `exit(code)` - exit program with specified exit code
- **Comments**: Line comments with `//` (e.g., `// this is a comment`)

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
- Uses Pratt parsing for proper operator precedence
- `LiteralExpr` - Represents number/string literals
- `IdentifierExpr` - Represents variable references (e.g., `x`, `myVar`)
- `BinaryExpr` - Binary expression with `Left`, `Op`, and `Right` fields
- `VarDeclStmt` - Variable declaration (e.g., `val x = 5`)
- `FunctionDecl` - Function declaration with name and body
- Outputs: `*Program` and `[]error`

**Semantic Analyzer** (`frontend/semantic/analyzer.go`):
- `Analyzer` - Performs type checking and semantic validation
- `Scope` - Lexical scope with parent pointer for variable lookup
- `Type` interface - Represents types in the type system (IntegerType, StringType, etc.)
- `TypedProgram` - AST annotated with type information
- Variable handling:
  - Tracks variable declarations in symbol table
  - Detects undefined variable references
  - Detects duplicate declarations in same scope
  - Type inference from initializer expression
- Type checking rules:
  - Arithmetic operators (`+`, `-`, `*`, `/`, `%`) require integer operands
  - Comparison operators (`==`, `!=`, `<`, `>`, `<=`, `>=`) require integer operands
  - Built-in functions are validated against their registered signatures
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

**Code Generator** (`backend/codegen/codegen.go`):
- `AsGenerator` interface with `Generate() (string, error)`
- `CodeGenContext` - Tracks variable stack offsets during code generation
- Generates ARM64 assembly targeting macOS
- Variable storage:
  - Variables stored on stack relative to frame pointer (x29)
  - 16-byte aligned stack slots
  - Proper handling of nested expressions with register spilling
- All operations store result in register `x2`
- Uses ARM64 instructions: `add`, `sub`, `mul`, `sdiv`, `cmp`, `cset`, `msub`, `str`, `ldr`, `stp`, `ldp`

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

All tests follow table-driven patterns with subtests using `t.Run()`. See `docs/TESTING.md` for detailed testing documentation.

**Semantic Analysis Tests** (`frontend/semantic/analyzer_test.go`):
- Type checking for all operators (arithmetic and comparison)
- Type error detection and reporting
- Multi-statement program analysis
- Integration with error framework

### End-to-End Tests

E2E tests live in the `test/` directory and read example files from `_examples/`:

- **`test/sl/e2e_test.go`** - Slang compiler e2e tests (reads `_examples/slang/*.sl`)
- **`test/slasm/e2e_test.go`** - Assembler e2e tests (reads `_examples/arm64/*.s`)
- **`test/testutil/`** - Shared expectation parsing utilities

Example files use `@test:` directives in header comments to specify expectations:

```slang
// @test: exit_code=42
// @test: requires_system_asm=true
fn main() {
    42
}
```

```asm
; @test: exit_code=5
; @test: skip=reason to skip
.global _start
...
```

Supported directives:
- `exit_code=N` - Expected exit code (default: 0)
- `stdout=text` - Expected stdout output
- `stderr=text` - Expected stderr output
- `skip=reason` - Skip the test with a reason
- `requires_system_asm=true` - Test requires system assembler (sl tests only)
- `expect_error=true` - Test expects a compilation error
- `error_stage=lexer|parser|semantic` - Which stage should produce the error
- `error_contains=text` - Error message should contain this text

### Integration Tests

`integration_test.go` contains pipeline integration tests:
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

3. **Code Generator** (`backend/codegen/codegen.go`):
   - Add case in `GenerateExpr()` switch statement with ARM64 instructions

4. **Tests**: Update test files for all three components

5. **E2E Tests**: Add example files to `_examples/slang/` with `@test:` directives

### Adding a New Built-in Function

Built-in functions are registered in a central registry and handled specially by the semantic analyzer and code generator.

1. **Registry** (`frontend/semantic/builtins.go`):
   - Add entry to the `Builtins` map with parameter types, return type, and flags

   ```go
   var Builtins = map[string]BuiltinFunc{
       "exit":  {ParamTypes: []Type{TypeI64}, ReturnType: TypeVoid, NoReturn: true},
       "print": {ParamTypes: []Type{TypeI64}, ReturnType: TypeVoid, NoReturn: false},
       // Add new built-in here
   }
   ```

2. **Code Generator - Typed** (`backend/codegen/typed_codegen.go`):
   - Add case in `generateBuiltinCall()` switch statement
   - Implement the generation function (e.g., `generateExitBuiltin()`)

3. **Code Generator - AST** (`backend/codegen/codegen.go`):
   - Add case in `generateBuiltinCallAST()` switch statement
   - Implement the generation function (e.g., `generateExitBuiltinAST()`)

4. **E2E Tests**: Add example files to `_examples/slang/builtins/` with `@test:` directives

**Example: The `exit()` built-in**

```slang
fn main(): void {
    exit(42)           // exit with literal
    exit(10 + 20)      // exit with expression
    val code = 7
    exit(code)         // exit with variable
}
```

Generated ARM64 assembly for `exit(42)`:
```asm
    mov x2, #42      // evaluate exit code into x2
    mov x0, x2       // move to x0 (syscall argument)
    mov x16, #1      // syscall 1 = exit
    svc #0           // invoke syscall
```

### Variable Syntax

Variables are declared using `val` (immutable) or `var` (mutable) keywords (Kotlin-style):

```slang
fn main(): void {
    val x = 42           // declare immutable x with value 42
    var y = 10           // declare mutable y with value 10
    val sum = x + y      // use variables in expressions
    print(sum)           // prints 52

    y = y + 5            // reassign mutable variable
    print(y)             // prints 15

    val result = x * 2 + y   // complex expressions work correctly
    print(result)            // prints 99
}
```

**Variable rules:**
- Variables must be declared before use
- `val` variables cannot be reassigned (immutable)
- `var` variables can be reassigned
- Variable names start with a letter, followed by letters, digits, or underscores
- Type is inferred from the initializer expression

**Error examples:**
```slang
print(x)       // Error: undefined variable 'x'
val x = 5
val x = 10     // Error: variable 'x' is already declared in this scope
x = 20         // Error: cannot assign to immutable variable 'x'
```

### Current Limitations

- No parentheses support in expressions
- No control flow (if/else, loops)
- print() requires system assembler (`--assembler system`)

## Module Information

- **Module Path**: `github.com/seanrogers2657/slang`
- **Go Version**: 1.24
- **Dependencies**:
  - `github.com/davecgh/go-spew` - Debug printing
  - `github.com/urfave/cli/v2` - CLI framework
- when committing, don't include claude's signature