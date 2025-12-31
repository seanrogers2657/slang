# Status

DRAFT, 2025-12-31

# Summary/Motivation

Add error handling to Slang using simple return values rather than exceptions. Functions can return errors as values that callers must explicitly handle or propagate, following the "errors are values" philosophy of Go.

# Goals/Non-Goals

- [goal] Errors as first-class values that can be stored, passed, and returned
- [goal] Simple syntax for returning errors from functions
- [goal] Explicit error handling at call sites
- [goal] Lightweight error propagation operator
- [goal] Custom error types via structs
- [goal] Integration with existing type system (nullable types, structs)
- [non-goal] Try/catch/throw exception mechanism
- [non-goal] Automatic error propagation (must be explicit)
- [non-goal] Stack traces in errors (reserved for panic, see SEP-5)
- [non-goal] Error hierarchies or inheritance
- [non-goal] Checked exceptions

# Design Decisions

These decisions should be validated during implementation:

1. **Error Representation**: Use a built-in `error` type that is either `nil` or contains a message
   - Simple and familiar from Go
   - Alternative: `Result<T, E>` type (more complex but safer)

2. **Multiple Returns**: Functions can return `(T, error)` tuples
   - Explicit about what can fail
   - Caller must handle both values

3. **Error Propagation**: Use `!` operator to propagate errors
   - `val x = tryThing()!` returns early if error
   - Lighter syntax than manual if-checks

4. **Error Creation**: Simple `error("message")` constructor
   - Returns an error value with the given message

# APIs

- `error` - Built-in type representing an error (nil or message)
- `error("message")` - Constructor to create an error value
- `nil` - Represents "no error" (success case)
- `!` - Error propagation operator (returns early if error)
- `.message` - Field to get error message string

# Description

## Core Concept

Errors in Slang are simple values. A function that can fail returns both its result and an error:

```slang
divide = (a: i64, b: i64) -> (i64, error) {
    if b == 0 {
        return (0, error("division by zero"))
    }
    (a / b, nil)
}
```

The caller must handle both the result and the error:

```slang
main = () {
    val (result, err) = divide(10, 2)
    if err != nil {
        print("Error: ")
        print(err.message)
        exit(1)
    }
    print(result)
}
```

## Step 1: Lexer Changes

**File:** `compiler/lexer/lexer.go`

Add tokens:

```go
TokenTypeError     // 'error' keyword
TokenTypeNil       // 'nil' keyword
TokenTypeBang      // '!' for error propagation
```

Add to keywords map:
```go
"error": TokenTypeError,
"nil":   TokenTypeNil,
```

## Step 2: AST Changes

**File:** `compiler/ast/ast.go`

Add tuple type for multiple returns:

```go
// TupleType represents (T1, T2, ...) for multiple return values
type TupleType struct {
    Elements []TypeExpr
}

// TupleExpr represents (expr1, expr2, ...) value construction
type TupleExpr struct {
    Elements []Expression
    Pos      Position
}

// TupleDestructure represents val (a, b) = expr
type TupleDestructure struct {
    Names []string
    Pos   Position
}

// ErrorExpr represents error("message") constructor
type ErrorExpr struct {
    Message Expression
    Pos     Position
}

// PropagateExpr represents expr! (error propagation)
type PropagateExpr struct {
    Expr Expression
    Pos  Position
}
```

## Step 3: Parser Changes

**File:** `compiler/parser/parser.go`

### Tuple Type Parsing

In return type position, allow parenthesized tuple types:

```slang
foo = () -> (i64, error) { ... }
```

### Tuple Expression Parsing

```slang
return (value, nil)
return (0, error("failed"))
```

### Tuple Destructuring

```slang
val (result, err) = someFunction()
```

### Error Constructor

```slang
error("message")
```

### Propagation Operator

```slang
val x = tryThing()!  // postfix !
```

## Step 4: Type System Changes

**File:** `compiler/semantic/types.go`

```go
// ErrorType is the built-in error type
type ErrorType struct{}

func (t ErrorType) String() string { return "error" }
func (t ErrorType) Equals(other Type) bool {
    _, ok := other.(ErrorType)
    return ok
}

// TupleType for multiple return values
type TupleType struct {
    Elements []Type
}

func (t TupleType) String() string {
    parts := make([]string, len(t.Elements))
    for i, elem := range t.Elements {
        parts[i] = elem.String()
    }
    return "(" + strings.Join(parts, ", ") + ")"
}

// NilType is the type of 'nil' (no error)
type NilType struct{}

func (t NilType) String() string { return "nil" }
```

### Type Rules

| Expression | Type |
|------------|------|
| `nil` | `NilType` (assignable to `error`) |
| `error("msg")` | `error` |
| `err.message` | `string` |
| `(a, b)` | `(T1, T2)` where a: T1, b: T2 |
| `expr!` | unwrapped type from tuple |

### Subtyping

- `NilType` is assignable to `error`
- `error` is assignable to `error`
- Tuples are covariant in their element types

## Step 5: Semantic Analysis

**File:** `compiler/semantic/analyzer.go`

### Error Constructor Validation

```go
func (a *Analyzer) analyzeErrorExpr(expr *ast.ErrorExpr) TypedExpression {
    // Message must be a string
    msgTyped := a.analyzeExpression(expr.Message)
    if !msgTyped.GetType().Equals(TypeString) {
        a.addError("error() requires string argument", expr.Pos)
    }
    return &TypedErrorExpr{Type: TypeError, Message: msgTyped}
}
```

### Tuple Destructuring Validation

```go
func (a *Analyzer) analyzeTupleDestructure(stmt *ast.VarDeclStmt) {
    // val (a, b) = expr
    initType := a.analyzeExpression(stmt.Init).GetType()
    tuple, ok := initType.(TupleType)
    if !ok {
        a.addError("cannot destructure non-tuple type", stmt.Pos)
        return
    }
    if len(stmt.Names) != len(tuple.Elements) {
        a.addError("tuple destructure count mismatch", stmt.Pos)
    }
    // Bind each name to its corresponding type
}
```

### Propagation Operator Validation

```go
func (a *Analyzer) analyzePropagateExpr(expr *ast.PropagateExpr) TypedExpression {
    innerTyped := a.analyzeExpression(expr.Expr)
    tuple, ok := innerTyped.GetType().(TupleType)
    if !ok || len(tuple.Elements) != 2 {
        a.addError("! operator requires (T, error) tuple", expr.Pos)
        return errorExpr
    }
    if !tuple.Elements[1].Equals(TypeError) {
        a.addError("! operator requires error as second element", expr.Pos)
        return errorExpr
    }
    // Check that current function returns error
    if !a.currentFunctionReturnsError() {
        a.addError("! operator can only be used in functions that return error", expr.Pos)
    }
    // Return type is the first element (unwrapped)
    return &TypedPropagateExpr{Type: tuple.Elements[0], Inner: innerTyped}
}
```

## Step 6: Code Generation

**File:** `compiler/codegen/typed_codegen.go`

### Error Type Representation

Errors are represented as a pointer to an error struct, or 0 for nil:

```
error = 0                   // nil (no error)
error = pointer to struct   // error with message
```

Error struct in memory:
```
offset 0: pointer to message string
```

### nil Generation

```asm
mov x2, #0              // nil is just 0
```

### Error Constructor Generation

```asm
// error("message")
// Allocate error struct (8 bytes)
// Store message pointer
// Return pointer to struct in x2
```

### Tuple Return Generation

For `return (value, err)`:

```asm
// Evaluate value into x0
// Evaluate error into x1
// Return (caller expects x0=value, x1=error)
```

### Propagation Operator Generation

For `val x = callThatReturnsError()!`:

```asm
    bl _fn_that_returns_error    // x0=value, x1=error
    cbnz x1, .Lpropagate_{id}    // if error != nil, propagate
    mov x2, x0                   // use value
    b .Lcontinue_{id}
.Lpropagate_{id}:
    mov x0, x1                   // return the error
    b .Lfn_epilogue              // early return
.Lcontinue_{id}:
```

## Step 7: Calling Convention

Functions returning `(T, error)` use:
- `x0` for the value (first tuple element)
- `x1` for the error (second tuple element)

This is efficient and natural for ARM64.

# Alternatives

1. **Result<T, E> type (Rust-style)**: More type-safe, forces handling. Rejected for simplicity - the extra type parameter adds complexity without much benefit for a simple language.

2. **Exceptions (try/catch)**: Traditional approach. Rejected because it hides control flow and makes it hard to reason about what can fail.

3. **Optional return only (Swift-style)**: Use `T?` for fallible functions. Rejected because it loses error information - you only know something failed, not why.

4. **Panic everywhere (SEP-5)**: Just panic on errors. Rejected because panics are for unrecoverable errors; many errors are expected and recoverable.

5. **Checked exceptions (Java-style)**: Force declaration of all errors. Rejected because it's verbose and leads to catch-all blocks.

6. **Error codes (C-style)**: Return integer codes. Rejected because it's error-prone and loses context.

# Testing

- **Lexer tests**: Token recognition for `error`, `nil`, `!`
- **Parser tests**:
  - Tuple type parsing `(T, error)`
  - Tuple expression parsing `(value, err)`
  - Tuple destructuring `val (a, b) = expr`
  - Error constructor `error("msg")`
  - Propagation operator `expr!`
- **Semantic tests**:
  - Error type checking
  - Propagation in non-error-returning functions (should error)
  - Tuple element count mismatches
- **Codegen tests**:
  - Error representation
  - Tuple returns
  - Propagation branching
- **E2E tests** in `_examples/slang/errors/`:
  - Basic error creation and handling
  - Error propagation with `!`
  - Nested function error handling
  - Nil checks

# Code Examples

## Example 1: Basic Error Handling

Demonstrates returning and handling errors.

```slang
divide = (a: i64, b: i64) -> (i64, error) {
    if b == 0 {
        return (0, error("division by zero"))
    }
    (a / b, nil)
}

main = () {
    val (result, err) = divide(10, 2)
    if err != nil {
        print("Error: ")
        print(err.message)
        exit(1)
    }
    print(result)  // prints: 5
}
```

## Example 2: Error Propagation

Demonstrates the `!` operator for concise error propagation.

```slang
parseNumber = (s: string) -> (i64, error) {
    // ... parsing logic
}

double = (s: string) -> (i64, error) {
    val n = parseNumber(s)!  // propagate error if parsing fails
    (n * 2, nil)
}

main = () {
    val (result, err) = double("21")
    if err != nil {
        print(err.message)
        exit(1)
    }
    print(result)  // prints: 42
}
```

## Example 3: Chained Error Propagation

Demonstrates propagating errors through multiple function calls.

```slang
readFile = (path: string) -> (string, error) {
    // ... file reading logic
}

parseConfig = (content: string) -> (Config, error) {
    // ... parsing logic
}

loadConfig = (path: string) -> (Config, error) {
    val content = readFile(path)!      // propagate file errors
    val config = parseConfig(content)! // propagate parse errors
    (config, nil)
}

main = () {
    val (config, err) = loadConfig("app.conf")
    if err != nil {
        print("Failed to load config: ")
        print(err.message)
        exit(1)
    }
    // use config...
}
```

## Example 4: Custom Error Information

Demonstrates creating errors with context.

```slang
validateAge = (age: i64) -> (i64, error) {
    if age < 0 {
        return (0, error("age cannot be negative"))
    }
    if age > 150 {
        return (0, error("age exceeds maximum"))
    }
    (age, nil)
}

main = () {
    val (age, err) = validateAge(-5)
    if err != nil {
        print(err.message)  // prints: age cannot be negative
    }
}
```

## Example 5: Ignoring Errors (Explicit)

Demonstrates explicitly ignoring errors when appropriate.

```slang
tryParse = (s: string) -> (i64, error) {
    // ...
}

main = () {
    // Explicitly ignore error by using _
    val (value, _) = tryParse("maybe")
    print(value)  // prints 0 if parsing failed
}
```

## Example 6: Error Handling in Loops

Demonstrates collecting errors during iteration.

```slang
processItem = (item: i64) -> (i64, error) {
    if item < 0 {
        return (0, error("negative item"))
    }
    (item * 2, nil)
}

main = () {
    val items = [1, 2, -3, 4, 5]
    var failed = 0

    for item in items {
        val (result, err) = processItem(item)
        if err != nil {
            failed = failed + 1
        } else {
            print(result)
        }
    }

    print("Failed: ")
    print(failed)
}
```

## Example 7: Returning Early on First Error

Demonstrates stopping on first error.

```slang
processAll = (items: Array<i64>) -> (i64, error) {
    var sum = 0
    for item in items {
        val processed = processItem(item)!  // return on first error
        sum = sum + processed
    }
    (sum, nil)
}
```

# Implementation Order

1. **Lexer** - Add `error`, `nil`, `!` tokens
2. **AST** - Add tuple types, expressions, destructuring, error nodes
3. **Parser** - Parse new syntax
4. **Types** - Add `ErrorType`, `TupleType`, `NilType`
5. **Semantic** - Type checking for errors and propagation
6. **Codegen** - Generate code for error handling
7. **E2E Tests** - Integration tests

# Files to Modify

| File | Changes |
|------|---------|
| `compiler/lexer/lexer.go` | Add `TokenTypeError`, `TokenTypeNil`, `TokenTypeBang` |
| `compiler/ast/ast.go` | Add `TupleType`, `TupleExpr`, `ErrorExpr`, `PropagateExpr` |
| `compiler/parser/parser.go` | Parse tuples, error(), destructuring, `!` |
| `compiler/semantic/types.go` | Add `ErrorType`, `TupleType`, `NilType` |
| `compiler/semantic/analyzer.go` | Error type checking, propagation validation |
| `compiler/codegen/typed_codegen.go` | Error and tuple code generation |

# Risks and Limitations

1. **No Error Wrapping**: Errors lose context as they propagate. Future enhancement could add `error("context", innerError)`.

2. **No Error Types**: All errors are the same type with just a message. Future enhancement could allow custom error structs.

3. **No Stack Traces**: Errors don't capture where they occurred. This is intentional - stack traces are for panics (SEP-5).

4. **Tuple Overhead**: Returning two values uses two registers. This is efficient on ARM64 but may impact other platforms.

5. **Must Handle Every Call**: No implicit error handling. This is intentional for explicitness but verbose.

# Future Enhancements

1. **Error Wrapping**
   ```slang
   val result = innerFunc()! wrap "outer context"
   ```

2. **Custom Error Types**
   ```slang
   FileError = struct {
       val path: string
       val code: i64
   }
   ```

3. **Pattern Matching on Errors**
   ```slang
   match err {
       FileNotFound => print("file not found")
       PermissionDenied => print("access denied")
       _ => print("unknown error")
   }
   ```

4. **Error Chaining**
   ```slang
   err.cause  // get underlying error
   ```
