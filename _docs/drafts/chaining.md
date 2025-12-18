# Function Chaining with Pipe Operator

**Status:** Draft
**Author:** Design Discussion
**Date:** December 2024

## Overview

This document describes the design for a pipe operator (`|>`) that enables function chaining in Slang. The feature transforms deeply nested function calls into a readable left-to-right flow.

```slang
// Before: nested calls (inside-out reading)
val result = double(add_one(double(5)))

// After: pipe chaining (left-to-right reading)
val result = 5 |> double |> add_one |> double
```

## Pipe Operator `|>`

### Basic Semantics

The pipe operator takes a value on the left and a function on the right, calling the function with that value.

```slang
value |> func        // equivalent to: func(value)
```

### Operator Properties

| Property | Value |
|----------|-------|
| Precedence | Lowest (below `\|\|`) |
| Associativity | Left-to-right |
| Operands | Left: any expression, Right: pipe target |

Precedence (lowest to highest):
```
|>              (pipe)
||              (logical or)
&&              (logical and)
== != < > <= >=  (comparison)
+ -             (additive)
* / %           (multiplicative)
! -             (unary)
```

### Chaining

Multiple pipes chain naturally due to left associativity:

```slang
5 |> double |> add_one |> triple
// Parses as: ((5 |> double) |> add_one) |> triple
// Executes: triple(add_one(double(5)))
```

## Pipe Target Forms

The right-hand side of `|>` can take several forms.

### Form 1: Bare Function Name (Implicit)

For single-parameter functions:

```slang
5 |> double              // → double(5)
"hello" |> print         // → print("hello")
```

### Form 2: Function Call with `it` (Explicit)

For multi-parameter functions, use `it` to mark where the piped value goes:

```slang
5 |> pow(it, 2)          // → pow(5, 2)
5 |> add(10, it)         // → add(10, 5)
5 |> clamp(0, it, 100)   // → clamp(0, 5, 100)
```

Multiple `it` references are allowed:

```slang
5 |> multiply(it, it)    // → multiply(5, 5) = 25
```

### Form 3: Lambda Expression

For inline transformations with a named parameter:

```slang
5 |> x -> double(x)
5 |> (x) -> double(x)
5 |> (x: i64) -> double(x)
5 |> (x: i64) -> { double(x) }
```

## The `it` Keyword

### Semantics

- `it` is a reserved keyword
- Only valid on the right-hand side of a `|>` operator
- Refers to the value from the immediately enclosing pipe's left-hand side
- Cannot be used as a variable name

### Scoping Rules

Each `|>` creates a scope where `it` is bound to the left-hand side value:

```slang
5 |> pow(it, 2)              // it = 5
5 |> double |> pow(it, 2)    // it = double(5) = 10
```

### Nested Pipes

In nested pipes, `it` refers to the innermost enclosing pipe:

```slang
5 |> a(it |> b |> c(it))
// Parses as: 5 |> a((it |> b) |> c(it))
//
// Outer pipe: it = 5
// Inner chain: it |> b → b(5)
//              b(5) |> c(it) where it = b(5) → c(b(5))
// Result: a(c(b(5)))
```

### `it` Not Available in Lambdas

When using lambda syntax, `it` is shadowed by the named parameter:

```slang
5 |> (x) -> double(x)        // ✓ OK - use x
5 |> (x) -> double(it)       // ✗ Error: 'it' not available, use 'x'
```

However, nested pipes inside a lambda create new `it` scopes:

```slang
5 |> (x) -> { x |> pow(it, 2) }    // ✓ OK - inner pipe has its own 'it'
// x = 5, inner it = 5
// Result: pow(5, 2) = 25
```

## Lambda Syntax

### Single Parameter Forms

Flexible syntax for single-parameter lambdas:

```slang
// Shortest - no parens, no type
x -> double(x)
x -> add(x, 10)
x -> { add(x, 10) }

// With parens, no type
(x) -> double(x)
(x) -> add(x, 10)

// With parens and type
(x: i64) -> double(x)
(x: i64) -> { double(x) }
```

### Bare Function in Lambda

A bare function name in lambda body is called with the parameter:

```slang
x -> double          // equivalent to: x -> double(x)
(x: i64) -> double   // equivalent to: (x: i64) -> double(x)
```

### Multi-Parameter Lambdas

Multiple parameters require parentheses and explicit types:

```slang
(x: i64, y: i64) -> add(x, y)
(a: i64, b: i64, c: i64) -> { a + b + c }
```

Invalid forms:

```slang
x, y -> add(x, y)              // ✗ Error: multi-param needs parens
(x, y) -> add(x, y)            // ✗ Error: multi-param needs types
(x: i64, y) -> add(x, y)       // ✗ Error: all params need types
```

### Lambda Syntax Summary

| Form | Parens | Types | Example |
|------|--------|-------|---------|
| Single | Optional | Optional | `x -> expr` |
| Single | Optional | Optional | `(x) -> expr` |
| Single | Required | Explicit | `(x: i64) -> expr` |
| Multiple | Required | Required | `(x: i64, y: i64) -> expr` |

## Builtins and Void

Piping to builtins works normally:

```slang
42 |> print              // prints 42
"hello" |> print         // prints hello
100 |> exit              // exits with code 100
```

Void-returning functions cannot be chained further:

```slang
42 |> print |> double    // ✗ Error: cannot pipe void to function
```

## Grammar

```ebnf
pipe_expr     = logic_or ("|>" pipe_target)* ;

pipe_target   = IDENTIFIER                                    (* bare function *)
              | IDENTIFIER "(" arguments ")"                  (* call with it *)
              | lambda ;

lambda        = IDENTIFIER "->" lambda_body                   (* x -> expr *)
              | "(" param_list ")" "->" lambda_body ;         (* (params) -> expr *)

param_list    = param
              | param ("," param)+ ;                          (* multi requires types *)

param         = IDENTIFIER                                    (* inferred type *)
              | IDENTIFIER ":" type ;                         (* explicit type *)

lambda_body   = IDENTIFIER                                    (* bare function *)
              | IDENTIFIER "(" arguments ")"                  (* function call *)
              | "{" expression "}" ;                          (* braced expression *)

arguments     = (argument ("," argument)*)? ;
argument      = expression ;                                  (* may contain 'it' *)
```

## Examples

### Basic Chaining

```slang
val result = 5 |> double |> add_one |> triple
// triple(add_one(double(5)))
```

### Multi-Argument Functions

```slang
val powered = 2 |> pow(it, 8)              // pow(2, 8) = 256
val clamped = 150 |> clamp(0, it, 100)     // clamp(0, 150, 100) = 100
```

### Multiple `it` Usage

```slang
val squared = 7 |> multiply(it, it)        // multiply(7, 7) = 49
```

### Lambda Transformations

```slang
val result = 5 |> x -> x * 2 |> y -> y + 1
// (5 * 2) + 1 = 11

val complex = 10 |> (x: i64) -> { x |> double |> pow(it, 2) }
// pow(double(10), 2) = pow(20, 2) = 400
```

### Nested Pipes

```slang
val result = 5 |> outer(it |> inner)
// outer(inner(5))

val result = 10 |> combine(it |> double, it |> triple)
// combine(double(10), triple(10)) = combine(20, 30)
```

### End with Builtin

```slang
100 |> double |> print
// prints 200
```

### Complex Expressions

```slang
val x = (5 |> double) + (3 |> triple)
// 10 + 9 = 19
```

## Error Messages

### Unknown Function

```
Error: unknown function 'foo'
  --> file.sl:1:8
  |
1 | 5 |> foo
  |      ^^^ function not found
```

### Arity Mismatch (Implicit Form)

```
Error: cannot pipe to 'add' - function takes 2 parameters
  --> file.sl:1:6
  |
1 | 5 |> add
  |      ^^^ expected 1 parameter
  |
  hint: use explicit form with 'it': 5 |> add(it, <arg>)
```

### Missing `it` in Call

```
Error: pipe with function call must use 'it' placeholder
  --> file.sl:1:6
  |
1 | 5 |> add(1, 2)
  |      ^^^^^^^^^ no 'it' found
  |
  hint: replace one argument with 'it': 5 |> add(it, 2)
```

### `it` Outside Pipe

```
Error: 'it' can only be used within a pipe expression
  --> file.sl:1:9
  |
1 | val x = it + 5
  |         ^^ not in pipe context
```

### `it` in Lambda

```
Error: 'it' is not available in lambda scope
  --> file.sl:1:18
  |
1 | 5 |> (x) -> add(x, it)
  |                    ^^ use 'x' instead
  |
  hint: lambda parameter 'x' shadows 'it'
```

### Type Mismatch

```
Error: cannot pipe string to 'double' which expects i64
  --> file.sl:1:1
  |
1 | "hello" |> double
  | ^^^^^^^ ── ^^^^^^ expects i64
  | │
  | this is string
```

### Multi-Param Lambda Missing Types

```
Error: multi-parameter lambda requires type annotations
  --> file.sl:1:6
  |
1 | 5 |> (x, y) -> add(x, y)
  |      ^^^^^^ add types: (x: i64, y: i64)
```

## Implementation Stages

### Stage 1: Lexer

Add new tokens:
- `TokenTypePipe` for `|>`
- `TokenTypeIt` for `it` keyword
- `TokenTypeArrow` for `->` (lambda)

### Stage 2: AST

Add new node types:
- `PipeExpr` - pipe expression with left value and right target
- `ItExpr` - reference to piped value
- `LambdaExpr` - lambda with parameters and body

### Stage 3: Parser

- Parse `|>` as lowest-precedence left-associative binary operator
- Parse `it` as primary expression
- Parse lambda syntax with all forms
- Desugar bare function `func` to `func(it)` in pipe context

### Stage 4: Semantic Analysis

- Maintain pipe context stack for `it` resolution
- Type check `it` against enclosing pipe's LHS type
- Validate implicit form targets single-param functions
- Validate explicit form contains at least one `it`
- Enforce lambda scoping rules (no `it` when param named)

### Stage 5: Code Generation

- Store piped value on stack when entering pipe
- Load from stack when generating `it`
- Generate function calls normally
- Handle nested pipes with stack of storage locations

## Open Questions

1. **Multi-param lambdas in pipes:** How should `(x: i64, y: i64) -> expr` work as a pipe target?
   - Option A: Not allowed as direct pipe target
   - Option B: Partial application (pipe fills first param)
   - Option C: Tuple destructuring

2. **Block expressions in lambdas:** Should `{ stmt; stmt; expr }` be allowed?
   - Currently only single expressions supported
   - Multi-statement blocks would require block expression semantics

3. **Method syntax:** Future consideration for `value.pipe(func)` style?
