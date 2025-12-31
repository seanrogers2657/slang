# Status

DRAFT, 2024-12-31

# Summary/Motivation

Add a pipe operator (`|>`) that enables function chaining in Slang. This transforms deeply nested function calls into a readable left-to-right flow, improving code clarity for data transformation pipelines.

```slang
// Before: nested calls (inside-out reading)
val result = double(add_one(double(5)))

// After: pipe chaining (left-to-right reading)
val result = 5 |> double |> add_one |> double
```

# Goals/Non-Goals

- [goal] Pipe operator `|>` for function chaining
- [goal] Implicit piping to single-parameter functions
- [goal] Explicit piping with `it` placeholder for multi-parameter functions
- [goal] Lambda expressions as pipe targets
- [goal] Clear error messages for pipe-related errors
- [non-goal] Method syntax (`value.pipe(func)`)
- [non-goal] Partial application
- [non-goal] Multi-statement block expressions in lambdas

# APIs

- `|>` - Pipe operator (lowest precedence, left-associative)
- `it` - Placeholder keyword for the piped value in explicit form
- `->` - Lambda arrow for inline transformations
- Bare function: `value |> func` desugars to `func(value)`
- Explicit form: `value |> func(it, arg)` places value where `it` appears
- Lambda form: `value |> x -> expr` binds piped value to `x`

# Description

## Step 1: Lexer Changes

Add new tokens:
- `TokenTypePipe` for `|>`
- `TokenTypeIt` for `it` keyword
- `TokenTypeArrow` for `->` (lambda)

## Step 2: AST Changes

Add new node types:

```go
// PipeExpr represents a pipe expression: left |> right
type PipeExpr struct {
    Left     Expr     // value being piped
    Pipe     Position // position of |>
    Target   PipeTarget // function, call with it, or lambda
}

// ItExpr represents the 'it' placeholder
type ItExpr struct {
    Pos Position
}

// LambdaExpr represents a lambda: (params) -> body
type LambdaExpr struct {
    Parameters []Parameter
    Arrow      Position
    Body       Expr
}
```

## Step 3: Parser Changes

Parse `|>` as lowest-precedence left-associative binary operator.

**Precedence (lowest to highest):**
```
|>              (pipe)
||              (logical or)
&&              (logical and)
== != < > <= >=  (comparison)
+ -             (additive)
* / %           (multiplicative)
! -             (unary)
```

**Pipe Target Forms:**

| Form | Syntax | Desugars To |
|------|--------|-------------|
| Bare function | `5 \|> double` | `double(5)` |
| Explicit with `it` | `5 \|> pow(it, 2)` | `pow(5, 2)` |
| Lambda | `5 \|> x -> x * 2` | `(x -> x * 2)(5)` |

**Lambda Syntax Variations:**

| Form | Example | Notes |
|------|---------|-------|
| Minimal | `x -> expr` | Single param, no parens |
| Parenthesized | `(x) -> expr` | Optional parens |
| Typed | `(x: i64) -> expr` | Explicit type |
| Multi-param | `(x: i64, y: i64) -> expr` | Requires parens and types |

## Step 4: Semantic Analysis

- Maintain pipe context stack for `it` resolution
- Type check `it` against enclosing pipe's LHS type
- Validate implicit form targets single-param functions
- Validate explicit form contains at least one `it`
- Enforce lambda scoping rules (`it` not available when lambda param shadows it)

**`it` Scoping Rules:**
- `it` is only valid on the right-hand side of a `|>` operator
- Each `|>` creates a new scope where `it` is bound to the left-hand side value
- In nested pipes, `it` refers to the innermost enclosing pipe
- Lambda parameters shadow `it` within the lambda body

## Step 5: Code Generation

- Store piped value on stack when entering pipe
- Load from stack when generating `it`
- Generate function calls normally
- Handle nested pipes with stack of storage locations

## Error Messages

The compiler provides clear error messages for pipe-related issues:

**Unknown function:**
```
Error: unknown function 'foo'
  --> file.sl:1:8
  |
1 | 5 |> foo
  |      ^^^ function not found
```

**Arity mismatch:**
```
Error: cannot pipe to 'add' - function takes 2 parameters
  --> file.sl:1:6
  |
1 | 5 |> add
  |      ^^^ expected 1 parameter
  |
  hint: use explicit form with 'it': 5 |> add(it, <arg>)
```

**Missing `it` in explicit form:**
```
Error: pipe with function call must use 'it' placeholder
  --> file.sl:1:6
  |
1 | 5 |> add(1, 2)
  |      ^^^^^^^^^ no 'it' found
  |
  hint: replace one argument with 'it': 5 |> add(it, 2)
```

**`it` outside pipe:**
```
Error: 'it' can only be used within a pipe expression
  --> file.sl:1:9
  |
1 | val x = it + 5
  |         ^^ not in pipe context
```

# Alternatives

1. **Method chaining (`.pipe()`)**: Requires methods on all types. Rejected because Slang doesn't have methods on primitive types.

2. **Elixir-style first-argument piping**: Always pipe into first argument position. Rejected because it's less flexible than `it` placeholder for multi-argument functions.

3. **No placeholder (`it`)**: Only support bare function piping. Rejected because it prevents piping to multi-argument functions.

4. **Underscore placeholder (`_`)**: Use `_` instead of `it`. Could work but `_` often means "discard" in pattern matching contexts.

5. **F#-style partial application**: Automatically curry functions. Too complex for MVP and changes function semantics.

# Testing

- **Lexer tests**: Token recognition for `|>`, `it`, `->`
- **Parser tests**: Pipe expression parsing, lambda parsing, precedence
- **Semantic tests**: `it` scoping, type checking, error detection
- **Codegen tests**: Correct assembly for pipe evaluation order

**E2E Test Files:**
- `pipe_basic.sl` - Basic single-function piping
- `pipe_chain.sl` - Multi-step pipe chains
- `pipe_explicit.sl` - Explicit form with `it` placeholder
- `pipe_lambda.sl` - Lambda expressions as pipe targets
- `pipe_nested.sl` - Nested pipe expressions
- `pipe_errors.sl` - Error case validation

# Code Examples

## Example 1: Basic Chaining

Demonstrates basic pipe chaining with single-parameter functions.

```slang
double = (x: i64) -> i64 { x * 2 }
add_one = (x: i64) -> i64 { x + 1 }

main = () {
    val result = 5 |> double |> add_one |> double
    print(result)  // prints: 22
    // Equivalent to: double(add_one(double(5)))
}
```

## Example 2: Explicit Form with `it`

Shows using `it` placeholder for multi-parameter functions.

```slang
pow = (base: i64, exp: i64) -> i64 {
    // ... power implementation
}

clamp = (min: i64, val: i64, max: i64) -> i64 {
    // ... clamp implementation
}

main = () {
    val powered = 2 |> pow(it, 8)           // pow(2, 8) = 256
    val clamped = 150 |> clamp(0, it, 100)  // clamp(0, 150, 100) = 100
    val squared = 7 |> multiply(it, it)     // multiply(7, 7) = 49
}
```

## Example 3: Lambda Transformations

Demonstrates inline transformations with lambda syntax.

```slang
main = () {
    val result = 5 |> x -> x * 2 |> y -> y + 1
    print(result)  // prints: 11

    val typed = 10 |> (x: i64) -> x * x
    print(typed)   // prints: 100
}
```

## Example 4: Nested Pipes

Shows nested pipe expressions with independent `it` scopes.

```slang
outer = (x: i64) -> i64 { x + 100 }
inner = (x: i64) -> i64 { x * 2 }
combine = (a: i64, b: i64) -> i64 { a + b }

main = () {
    val result1 = 5 |> outer(it |> inner)
    // outer(inner(5)) = outer(10) = 110

    val result2 = 10 |> combine(it |> double, it |> triple)
    // combine(double(10), triple(10)) = combine(20, 30) = 50
}
```

## Example 5: Ending with Builtin

Demonstrates piping to void-returning builtins.

```slang
double = (x: i64) -> i64 { x * 2 }

main = () {
    100 |> double |> print  // prints: 200
}
```

## Example 6: Error Case - Piping Void

Demonstrates compile error when trying to continue after void.

```slang
// @test: expect_error=true
// @test: error_contains=cannot pipe void
main = () {
    42 |> print |> double  // Error: cannot pipe void to function
}
```
