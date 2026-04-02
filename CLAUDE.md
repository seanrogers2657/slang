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

## Important Rules

- **Never delete files without permission**: Do not use `rm`, `git rm`, or any other file deletion commands without explicit user approval. Always ask before removing any files or directories.

## Naming Conventions

Slang code follows these naming conventions:

1. **Classes and Structs**: Use `CapitalCamelCase`
   - Examples: `Point`, `Counter`, `GraphNode`, `MathUtils`, `TreeNode`

2. **Methods and Functions**: Use `lower_snake_case`
   - Examples: `get_value`, `set_x`, `compute_distance`, `is_empty`, `add_two`
   - This applies to both static methods and instance methods
   - This applies to free functions as well

3. **Variables**: Use `lower_snake_case`
   - Examples: `my_value`, `count`, `node_1`, `left_val`, `total_sum`

4. **Fields**: Use `lower_snake_case`
   - Examples: `x`, `y`, `value`, `count`, `top_left`, `bottom_right`, `last_value`

**Examples:**

```slang
// CapitalCamelCase for class name
Counter = class {
    var count: s64

    // lower_snake_case for methods
    get_count = (self: &Counter) -> s64 {
        return self.count
    }

    add_amount = (self: &&Counter, amount: s64) {
        self.count = self.count + amount
    }
}

// lower_snake_case for free functions and variables
compute_distance = (p1: &Point, p2: &Point) -> s64 {
    val delta_x = p2.get_x() - p1.get_x()
    val delta_y = p2.get_y() - p1.get_y()
    return delta_x + delta_y
}
```

## Project Overview

**Slang** is a compiler for a simple programming language written in Go. It targets ARM64 assembly for macOS. The compiler follows a five-stage pipeline:

1. **Lexer** (`compiler/lexer`) - Tokenizes source code into tokens
2. **Parser** (`compiler/parser`) - Builds an Abstract Syntax Tree (AST) from tokens
3. **Semantic Analyzer** (`compiler/semantic`) - Performs type checking and semantic analysis
4. **IR Generator** (`compiler/ir`) - Converts typed AST to SSA-based Intermediate Representation
5. **ARM64 Backend** (`compiler/ir/backend/arm64`) - Generates ARM64 assembly from IR

The compiler currently supports:
- **Variables**: Immutable (`val`) and mutable (`var`) variables (e.g., `val x = 5`, `var y = 10`)
- **Types**:
  - Signed integers: `s8`, `s16`, `s32`, `s64`, `s128` (`int` is alias for `s64`)
  - Unsigned integers: `u8`, `u16`, `u32`, `u64`, `u128`
  - Other primitives: `bool`, `string`, `void`
  - Arrays: `[1, 2, 3]` with index access and `len()`
  - Structs: User-defined types with `val`/`var` fields
  - Nullable types: `T?` (e.g., `s64?`) with `null` value
  - Pointer types: `*T` (owned), `&T` (immutable borrow), `&&T` (mutable borrow)
- **Expressions**: Binary and unary expressions
- **Operators**:
  - Arithmetic: `+`, `-`, `*`, `/`, `%`
  - Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=` (return `bool`)
  - Logical: `&&` (and), `||` (or), `!` (not)
  - Field access: `.` for struct fields
  - Index access: `[]` for arrays
  - Safe navigation: `?.` for nullable field access
- **Boolean literals**: `true`, `false`
- **Statements**: Expression statements, variable declarations, assignments
- **Control Flow**:
  - `if`/`else` statements and expressions
  - `while` loops
  - `for` loops (C-style: `for (var i = 0; i < 10; i = i + 1) { }`)
  - `break` and `continue`
  - `when` expressions (conditional branching)
- **Functions**: Function declarations (e.g., `main = () { ... }`, `add = (a: int, b: int) -> int { ... }`)
- **Built-in Functions**:
  - `print(value)` - print a value to stdout (accepts `s64`, `string`, or `bool`)
  - `exit(code)` - exit program with specified exit code
  - `len(array)` - get array length
  - `sleep(nanoseconds)` - sleep for specified duration
  - `assert(condition, message)` - if condition is false, prints message to stderr and exits with code 1
  - `new value` - allocate a value on the heap (returns `*T`)
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
./sl run _examples/slang/arithmetic/add.sl
./sl run _examples/slang/arithmetic/subtract.sl
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

# Run specific component tests (using go test directly)
go test ./compiler/lexer/...             # Frontend lexer only
go test ./compiler/parser/...            # Frontend parser only
go test ./compiler/semantic/...          # Semantic analyzer only
go test ./compiler/ir/...                # IR generator and backend tests
go test ./test/sl/...                    # End-to-end integration tests

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
IR Generator (SSA-based intermediate representation)
    ↓
ARM64 Backend (code generation)
    ↓
Assembly Output (.s file)
    ↓
Assembler (slasm) → Executable binary
    ↓
[Optional: Execute] (run command only)
```

### Key Data Structures

**Lexer** (`compiler/lexer/lexer.go`):
- `Token` - Represents a single token with `Type` and `Value`
- `TokenType` - Enum for token types (Integer, Plus, Minus, etc.)
- Outputs: `[]Token` and `[]error`

**Parser** (`compiler/parser/parser.go`):
- Uses Pratt parsing for proper operator precedence
- `LiteralExpr` - Represents number/string literals
- `IdentifierExpr` - Represents variable references (e.g., `x`, `myVar`)
- `BinaryExpr` - Binary expression with `Left`, `Op`, and `Right` fields
- `VarDeclStmt` - Variable declaration (e.g., `val x = 5`)
- `FunctionDecl` - Function declaration with name and body
- Outputs: `*Program` and `[]error`

**Semantic Analyzer** (`compiler/semantic/analyzer.go`):
- `Analyzer` - Performs type checking and semantic validation
- `Scope` - Lexical scope with parent pointer for variable lookup
- `Type` interface - Represents types in the type system (IntegerType, BooleanType, StringType, etc.)
- `TypedProgram` - AST annotated with type information
- Variable handling:
  - Tracks variable declarations in symbol table
  - Detects undefined variable references
  - Detects duplicate declarations in same scope
  - Type inference from initializer expression
- Type checking rules:
  - Arithmetic operators (`+`, `-`, `*`, `/`, `%`) require integer operands
  - Comparison operators (`==`, `!=`, `<`, `>`, `<=`, `>=`) require integer operands and return `bool`
  - Logical operators (`&&`, `||`) require boolean operands and return `bool`
  - Unary `!` requires a boolean operand and returns `bool`
  - Built-in functions are validated against their registered signatures
- Outputs: `[]*CompilerError` and `*TypedProgram`

**Error Framework** (`errors/`):
- `CompilerError` - Rich error type with filename, position, and error span
- `FormatError()` - Formats errors with source code context and color highlighting
- Beautiful error messages showing:
  - Error type and message
  - File location (file:line:column)
  - Source code snippet
  - Visual error pointer (^ under the error)
  - Optional hints for fixing the error
  - Summary of total errors/warnings

**IR Generator** (`compiler/ir/generator.go`):
- Converts typed AST to SSA (Static Single Assignment) form
- Uses the "Simple and Efficient Construction of SSA Form" algorithm
- Handles phi node insertion for variables that join at control flow merge points
- Block sealing to complete phi node operands when all predecessors are known
- IR types: `IntType`, `BoolType`, `PtrType`, `StructType`, `ArrayType`, `NullableType`
- IR operations: `OpAdd`, `OpSub`, `OpMul`, `OpDiv`, `OpMod`, `OpLoad`, `OpStore`, `OpCall`, `OpPhi`, etc.

**ARM64 Backend** (`compiler/ir/backend/arm64/backend.go`):
- Generates ARM64 assembly from IR
- Simple stack-based register allocation
- Function prologue/epilogue with proper frame pointer management
- Phi node handling at control flow edges
- Runtime checks for:
  - Division/modulo by zero
  - Array bounds checking
  - Signed/unsigned integer overflow detection
- Stack trace printing on panic
- Deep copy for nested owned pointers

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

**Logical operations** (with short-circuit evaluation):
- Logical NOT (`!`): `cmp x2, #0` + `cset x2, eq`
- Logical AND (`&&`): Evaluates left, uses `cbz` to skip right if false
- Logical OR (`||`): Evaluates left, uses `cbnz` to skip right if true

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

The project has comprehensive test coverage across all compiler stages:
- Lexer, Parser, Semantic Analyzer - unit tests in respective packages
- IR Generator - unit tests in `compiler/ir/generator_test.go`
- ARM64 Backend - unit tests in `compiler/ir/backend/arm64/backend_test.go`
- End-to-End - 275+ tests covering all language features

All tests follow table-driven patterns with subtests using `t.Run()`. See `docs/TESTING.md` for detailed testing documentation.

**Semantic Analysis Tests** (`compiler/semantic/analyzer_test.go`):
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
main = () {
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

When adding a new operator, you must update four files:

1. **Lexer** (`compiler/lexer/lexer.go`):
   - Add `TokenType<OperatorName>` constant to the enum
   - Add parsing logic in the `Parse()` method

2. **Parser** (`compiler/parser/parser.go`):
   - Add case in `ParseBinaryExpression()` switch statement

3. **IR Generator** (`compiler/ir/generator.go`):
   - Add case in `generateBinaryExpr()` to emit the appropriate IR operation

4. **ARM64 Backend** (`compiler/ir/backend/arm64/backend.go`):
   - Add case in the operation switch to generate ARM64 instructions

5. **Tests**: Update test files for all components

6. **E2E Tests**: Add example files to `_examples/slang/` with `@test:` directives

### Adding a New Built-in Function

Built-in functions are registered in a central registry and handled specially by the semantic analyzer and IR generator.

1. **Registry** (`compiler/semantic/builtins.go`):
   - Add entry to the `Builtins` map with parameter types, return type, and flags

   ```go
   var Builtins = map[string]BuiltinFunc{
       "exit":  {ParamTypes: []Type{TypeS64}, ReturnType: TypeVoid, NoReturn: true},
       "print": {ParamTypes: []Type{TypeS64}, ReturnType: TypeVoid, AcceptedTypes: map[int][]Type{0: {TypeS64, TypeString, TypeBoolean}}},
       "len":   {ParamTypes: []Type{TypeError}, ReturnType: TypeS64, IsArrayLen: true},
       "sleep": {ParamTypes: []Type{TypeS64}, ReturnType: TypeVoid},
       // Add new built-in here
   }
   ```

2. **IR Generator** (`compiler/ir/generator.go`):
   - Add case in `generateBuiltinCall()` switch statement
   - Implement the IR generation for the builtin (e.g., `OpExit`, `OpPrint`)

3. **ARM64 Backend** (`compiler/ir/backend/arm64/backend.go`):
   - Add case in the operation switch to generate ARM64 instructions/syscalls

4. **E2E Tests**: Add example files to `_examples/slang/builtins/` with `@test:` directives

**Example: The `exit()` built-in**

```slang
main = () {
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

### Function Syntax

Functions are declared using the assignment syntax with optional return type:

```slang
// Function with no parameters, void return (omit return type)
main = () {
    print(42)
}

// Function with parameters and return type
add = (a: int, b: int) -> int {
    return a + b
}

// Function can use implicit return (last expression)
square = (x: int) -> int {
    x * x
}
```

### Struct Syntax

Structs are declared at the top level using the assignment syntax with `struct` keyword. Each field must have a `val` (immutable) or `var` (mutable) prefix:

```slang
// Define a struct with immutable and mutable fields
Point = struct {
    val x: s64    // immutable field
    var y: s64    // mutable field
}

// Nested structs
Rectangle = struct {
    val top_left: Point
    val bottom_right: Point
}

main = () {
    // Create struct instances with positional arguments
    // Note: no space between struct name and opening brace
    val p = Point{ 10, 20 }
    print(p.x)  // prints 10
    print(p.y)  // prints 20

    // Mutable fields can be reassigned
    p.y = 25
    print(p.y)  // prints 25

    // Immutable fields cannot be reassigned
    // p.x = 30  // Error: cannot assign to immutable field 'x'

    // Create with named arguments (any order)
    val q = Point{ y: 5, x: 3 }
    print(q.x)  // prints 3

    // Anonymous struct literal with type annotation
    val r: Point = { x: 7, y: 8 }
    print(r.x)  // prints 7

    // Nested struct creation
    val rect = Rectangle{ Point{ 0, 0 }, Point{ 100, 100 } }
    print(rect.top_left.x)       // prints 0
    print(rect.bottom_right.x)   // prints 100
}
```

**Struct rules:**
- Each field must be prefixed with `val` (immutable) or `var` (mutable)
- `val` fields cannot be reassigned after struct creation
- `var` fields can be reassigned
- Field names must be unique within a struct
- Structs must be declared at the top level (not inside functions)
- Struct names must be unique (no duplicate definitions)
- Struct literals use braces: `Point{ 1, 2 }` (no space between name and `{`)
- Anonymous struct literals require type annotation: `val p: Point = { x: 1, y: 2 }`

### Pointer Syntax (*T, &T, &&T)

Slang uses an ownership-based memory model with three pointer types:

- `*T` - Owned pointer with unique ownership (move semantics)
- `&T` - Immutable borrowed reference (read-only access)
- `&&T` - Mutable borrowed reference (can mutate var fields)

**Key principle**: `val`/`var` controls **reassignability** only. `&T` vs `&&T` controls **borrow mutability**.

```slang
Point = struct {
    var x: s64
    var y: s64
}

// Allocate on the heap with new 
main = () {
    val p = new Point{ 10, 20 }  // p: *Point
    print(p.x)  // Auto-dereference: prints 10
    print(p.y)  // prints 20

    // val binding can still mutate var fields (val only prevents reassignment)
    p.x = 100
    print(p.x)  // prints 100
}
```

**Ownership transfer (move semantics):**
```slang
// Passing *T to a function transfers ownership
consume_point = (p: *Point) -> s64 {
    return p.x + p.y
}

// Returning *T transfers ownership to caller
create_point = (x: s64, y: s64) -> *Point {
    return new Point{ x, y }
}

main = () {
    val p = create_point(10, 20)
    val sum = consume_point(p)  // p is moved
    // print(p.x)  // Error: p was moved
}
```

**Borrowing with &T and &&T:**
```slang
// &T borrows without taking ownership (read-only)
print_point = (p: &Point) {
    print(p.x)
    print(p.y)
}

// &&T allows mutation through the reference
double_x = (p: &&Point) {
    p.x = p.x * 2
}

main = () {
    val p = new Point{ 10, 20 }
    print_point(p)  // Auto-borrow: *Point -> &Point
    print(p.x)     // p still usable: prints 10

    double_x(p)     // Mutable borrow (val binding can still borrow as &&T)
    print(p.x)     // prints 20
}
```

**Deep copy with .copy():**
```slang
main = () {
    val p = new Point{ 10, 20 }
    val q = p.copy()  // Creates independent deep copy

    p.x = 100
    print(q.x)  // prints 10 (unchanged)
}
```

**Pointer rules:**
- `*T` values are move-only (assignment moves, not copies)
- Use `.copy()` to create an explicit deep copy
- `&T` and `&&T` can only appear in function parameter position
- `&T` is read-only; `&&T` can mutate var fields
- Auto-borrow: `*T` automatically converts to `&T` or `&&T` when passed to functions
- `val`/`var` only controls reassignability, not mutation through the pointer
- Multiple immutable borrows are allowed; multiple mutable borrows are not
- Memory is automatically freed when owned pointers go out of scope

### Variable Syntax

Variables are declared using `val` (immutable) or `var` (mutable) keywords (Kotlin-style):

```slang
main = () {
    val x = 42           // declare immutable x with value 42
    var y = 10           // declare mutable y with value 10
    val sum = x + y      // use variables in expressions
    print(sum)           // prints 52

    y = y + 5            // reassign mutable variable
    print(y)             // prints 15

    val result = x * 2 + y   // complex expressions work correctly
    print(result)            // prints 99

    // Boolean variables
    val is_valid = true
    val is_greater = x > y    // comparison returns bool
    print(is_valid && is_greater)  // prints "true" or "false"

    // Explicit type annotations
    val a: s16 = 1000        // explicitly typed as s16
    val b: s32 = 50000       // explicitly typed as s32
    val flag: bool = true    // explicitly typed as bool
}
```

**Variable rules:**
- Variables must be declared before use
- `val` variables cannot be reassigned (immutable)
- `var` variables can be reassigned
- Variable names start with a letter, followed by letters, digits, or underscores
- Type is inferred from the initializer expression, or can be explicitly annotated with `: Type`

**Error examples:**
```slang
print(x)       // Error: undefined variable 'x'
val x = 5
val x = 10     // Error: variable 'x' is already declared in this scope
x = 20         // Error: cannot assign to immutable variable 'x'
```

### Control Flow

Slang supports standard control flow constructs:

**If/Else Statements:**
```slang
main = () {
    val x = 10
    if x > 5 {
        print(1)
    }

    if x < 5 {
        print(0)
    } else {
        print(1)
    }

    // Else-if chains
    if x < 0 {
        print(-1)
    } else if x == 0 {
        print(0)
    } else {
        print(1)
    }
}
```

**If Expressions** (returns a value):
```slang
main = () {
    val x = 10
    val result = if x > 5 { 1 } else { 0 }  // result = 1
    print(result)

    // If expressions require else branch and matching types
    val sign = if x < 0 { -1 } else if x == 0 { 0 } else { 1 }
}
```

**While Loops:**
```slang
main = () {
    var i = 0
    while i < 5 {
        print(i)
        i = i + 1
    }
    // prints: 0 1 2 3 4
}
```

**For Loops** (C-style syntax with parentheses):
```slang
main = () {
    for (var i = 0; i < 5; i = i + 1) {
        print(i)
    }
    // prints: 0 1 2 3 4
}
```

**Break and Continue:**
```slang
main = () {
    var i = 0
    while true {
        if i >= 5 {
            break
        }
        if i % 2 == 0 {
            i = i + 1
            continue
        }
        print(i)
        i = i + 1
    }
    // prints: 1 3
}
```

### When Expressions

`when` expressions provide conditional branching similar to Kotlin's `when`:

```slang
main = () {
    val x = 5
    when {
        x > 10 -> exit(0)
        x > 3 -> exit(10)
        else -> exit(20)
    }
    // exits with code 10
}

// When as expression (returns a value)
get_value = (x: s64) -> s64 {
    return when {
        x < 0 -> -1
        x == 0 -> 0
        else -> 1
    }
}

// When with blocks
main = () {
    val x = 5
    when {
        x > 10 -> {
            print(1)
            exit(0)
        }
        else -> {
            print(0)
            exit(1)
        }
    }
}
```

**When rules:**
- `when` expressions require an `else` branch when used as expression
- All branches must return compatible types when used as expression
- Conditions are evaluated top-to-bottom, first match wins

### Array Syntax

Arrays are fixed-size collections with type inference:

```slang
main = () {
    // Array literal
    val arr = [1, 2, 3]

    // Index access (0-based)
    print(arr[0])  // prints 1
    print(arr[1])  // prints 2
    print(arr[2])  // prints 3

    // Array length
    print(len(arr))  // prints 3

    // Mutable array elements (requires var)
    var nums = [10, 20, 30]
    nums[0] = 100
    print(nums[0])  // prints 100

    // Arrays in loops
    for (var i = 0; i < len(arr); i = i + 1) {
        print(arr[i])
    }

    // Boolean arrays
    val flags = [true, false, true]
    print(flags[0])  // prints true
}
```

**Array rules:**
- Array type is inferred from element types (all elements must have same type)
- Index access is bounds-checked at runtime
- Negative indices cause runtime error
- `val` arrays cannot have elements reassigned; use `var` for mutable arrays

### Nullable Types

Slang supports nullable types with the `T?` syntax (Kotlin-style):

```slang
main = () {
    // Nullable variable with null
    val x: s64? = null

    // Nullable variable with value
    val y: s64? = 42

    // Null comparison
    print(x == null)  // prints true
    print(y == null)  // prints false
    print(y != null)  // prints true

    // Nullable struct fields
    Point = struct {
        val x: s64?
        val y: s64?
    }

    val p = Point{ null, 42 }
    print(p.y == null)  // prints false
}

// Functions can return nullable types
find_value = (x: s64) -> s64? {
    if x > 0 {
        return x
    }
    return null
}

main = () {
    val result = find_value(-5)
    if result == null {
        print(0)
    }
}
```

**Nullable rules:**
- Only nullable types (`T?`) can hold `null`
- Cannot assign `null` to non-nullable type
- Nested nullables (`T??`) are not allowed
- Use null comparison (`x == null`, `x != null`) for null checks

## Module Information

- **Module Path**: `github.com/seanrogers2657/slang`
- **Go Version**: 1.24
- **Dependencies**:
  - `github.com/davecgh/go-spew` - Debug printing
  - `github.com/urfave/cli/v2` - CLI framework
- when committing, don't include claude's signature