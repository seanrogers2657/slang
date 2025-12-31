# Status
DRAFT, 2025-12-31

# Summary/Motivation

Add a compile-time macro system to Slang that enables code generation and pattern-based transformations, reducing boilerplate and enabling domain-specific abstractions without runtime cost.

# Goals/Non-Goals

- [goal] Declarative macro definitions with pattern matching and substitution
- [goal] Macro invocation syntax that is visually distinct from function calls
- [goal] Hygienic macro expansion to prevent accidental variable capture
- [goal] Compile-time expansion with clear error messages pointing to macro source
- [goal] Support for expression macros, statement macros, and declaration macros
- [goal] Variadic arguments (repetition patterns) for flexible macro signatures
- [non-goal] Procedural macros (arbitrary compile-time code execution)
- [non-goal] Macro import/export across files (module system dependency)
- [non-goal] Recursive macro definitions in initial implementation
- [non-goal] Conditional compilation (`#ifdef` style)

# APIs

- `macro` keyword - Introduces a macro definition at the top level.
- `macro_name!(args)` - Invokes a macro with the `!` suffix distinguishing it from function calls.
- `$ident` - Captures an identifier in a macro pattern.
- `$expr` - Captures an expression in a macro pattern.
- `$stmt` - Captures a statement in a macro pattern.
- `$type` - Captures a type in a macro pattern.
- `$($pat),*` - Repetition pattern for zero or more comma-separated items.
- `$($pat),+` - Repetition pattern for one or more comma-separated items.

# Description

## Step 1: Lexer Changes

Add token support for macro syntax:
- Add `macro` keyword token (`TokenTypeMacro`)
- Add `$` token for pattern variables (`TokenTypeDollar`)
- Recognize `identifier!` as macro invocation (peek for `!` after identifier)
- Add `*` and `+` as repetition modifiers in pattern context

New tokens:
```go
TokenTypeMacro    // macro keyword
TokenTypeDollar   // $ for pattern variables
TokenTypeBang     // ! for macro invocation (already exists as NOT)
```

## Step 2: AST Changes

Add new AST nodes for macro definitions and invocations:

```go
// MacroPatternElement represents one element in a macro pattern
type MacroPatternKind int
const (
    MacroPatternLiteral MacroPatternKind = iota  // literal token
    MacroPatternCapture                           // $name:kind
    MacroPatternRepeat                            // $(...),*
)

type MacroPatternElement struct {
    Kind       MacroPatternKind
    Token      Token             // for literals
    Name       string            // capture name (e.g., "x" from $x)
    CaptureKind string           // "expr", "stmt", "ident", "type"
    Separator  string            // "," or "" for repetitions
    RepeatKind string            // "*" or "+"
    Inner      []MacroPatternElement // nested pattern for repetitions
}

// MacroRule represents one pattern => body rule in a macro
type MacroRule struct {
    Pattern []MacroPatternElement
    Body    []MacroPatternElement  // template with substitutions
}

// MacroDecl represents a macro definition
type MacroDecl struct {
    MacroKeyword Position
    Name         string
    NamePos      Position
    Rules        []MacroRule  // multiple rules for pattern matching
}

// MacroInvocation represents a macro call site
type MacroInvocation struct {
    Name       string
    NamePos    Position
    Bang       Position         // position of '!'
    LeftParen  Position
    Arguments  []Token          // raw tokens for pattern matching
    RightParen Position
}
```

## Step 3: Parser Changes

### Parsing Macro Definitions

Parse `macro name { pattern => body, ... }` syntax:

```slang
macro swap {
    ($a:ident, $b:ident) => {
        val temp = $a
        $a = $b
        $b = temp
    }
}
```

The parser must:
1. Parse the `macro` keyword
2. Parse the macro name
3. Parse one or more rules, each with pattern and body
4. Store raw token patterns (not full AST) for pattern matching

### Parsing Macro Invocations

When an identifier is followed by `!`, parse as macro invocation:

```slang
swap!(x, y)           // invoke swap macro
debug_print!(value)   // invoke debug_print macro
```

The parser collects raw tokens between parentheses and stores them for later expansion.

## Step 4: Macro Expansion Phase

Add a new compiler phase between parsing and semantic analysis:

```
Source → Lexer → Parser → [Macro Expansion] → Semantic → Codegen
```

The expander:
1. Collects all macro definitions from the AST
2. Finds all macro invocations
3. Pattern-matches invocation arguments against macro rules
4. Substitutes captures into the body template
5. Replaces the invocation with expanded AST nodes
6. Handles hygiene by renaming introduced variables

### Hygiene

Variables introduced by a macro get unique names to prevent capture:

```slang
macro swap {
    ($a:ident, $b:ident) => {
        val temp = $a    // temp becomes __macro_swap_temp_1
        $a = $b
        $b = temp
    }
}

main = () {
    var x = 1
    var y = 2
    var temp = 100       // user's temp, unaffected
    swap!(x, y)          // expands with __macro_swap_temp_1
    print(temp)          // prints 100, not affected
}
```

### Pattern Matching

Patterns match against token sequences:

| Pattern | Matches | Captures |
|---------|---------|----------|
| `$x:expr` | Any expression | The expression |
| `$x:ident` | An identifier | The identifier name |
| `$x:stmt` | A statement | The statement |
| `$x:type` | A type name | The type |
| `foo` | Literal `foo` | Nothing |
| `$($x:expr),*` | Zero or more comma-separated exprs | List of exprs |
| `$($x:expr),+` | One or more comma-separated exprs | List of exprs |

## Step 5: Error Handling

Macro errors should provide context showing both:
1. Where the macro was invoked
2. What rule failed to match

```
Error: macro pattern mismatch
  --> file.sl:10:5
   |
10 |     swap!(x)
   |     ^^^^^^^^ macro invocation here
   |
  note: macro 'swap' expects 2 identifiers
  --> file.sl:1:1
   |
 1 | macro swap {
 2 |     ($a:ident, $b:ident) => { ... }
   |      ^^^^^^^^^^^^^^^^^^^^^ pattern requires 2 arguments
```

## Step 6: Built-in Macros

Provide useful built-in macros:

### `assert!(condition)` / `assert!(condition, message)`

```slang
assert!(x > 0)
assert!(x > 0, "x must be positive")
```

Expands to:
```slang
if !x > 0 {
    print("Assertion failed: x > 0")
    exit(1)
}
```

### `debug!(expr)`

```slang
debug!(x + y)
```

Expands to:
```slang
{
    val __result = x + y
    print("x + y = ")
    print(__result)
    __result
}
```

### `todo!()`

```slang
unimplemented = () -> i64 {
    todo!()  // panics with "not yet implemented"
}
```

### `vec![a, b, c]` (future, with arrays)

```slang
val nums = vec![1, 2, 3, 4, 5]
```

## Quick Reference

| Syntax | Meaning | Example |
|--------|---------|---------|
| `macro name { ... }` | Define macro | `macro swap { ... }` |
| `name!(args)` | Invoke macro | `swap!(x, y)` |
| `$x:expr` | Capture expression | `$val:expr` |
| `$x:ident` | Capture identifier | `$name:ident` |
| `$x:stmt` | Capture statement | `$s:stmt` |
| `$($x),*` | Zero or more | `$($e:expr),*` |
| `$($x),+` | One or more | `$($e:expr),+` |

# Alternatives

1. **C-style preprocessor macros**: Simple text substitution without hygiene. Rejected because it leads to subtle bugs and poor error messages.

2. **Rust-style procedural macros**: Arbitrary compile-time Rust code. Rejected for MVP because it requires embedding a runtime/interpreter.

3. **Lisp-style macros**: Direct AST manipulation. More powerful but requires homoiconic syntax which Slang doesn't have.

4. **Template-based generics only**: No macros, just generics. Rejected because it doesn't support code generation patterns like `assert!` or `debug!`.

5. **Julia-style generated functions**: Staged programming. Too complex for MVP.

6. **Function-like syntax without `!`**: Could confuse macros with functions. The `!` suffix makes macro invocations visually distinct, signaling that compile-time expansion occurs.

# Testing

- **Lexer tests**: Token recognition for `macro`, `$`, `!` after identifier
- **Parser tests**: Macro definition parsing, invocation parsing, pattern syntax
- **Expansion tests**: Pattern matching, substitution, hygiene
  - Single-rule macros
  - Multi-rule macros with pattern matching
  - Repetition patterns (`$(...),*`)
  - Hygiene (no variable capture)
  - Nested macro invocations
- **Integration tests**: Full programs using macros
- **Error tests**: Clear error messages for mismatched patterns, undefined macros
- **E2E tests**: Add example files to `_examples/slang/macros/` with `@test:` directives

# Code Examples

## Example 1: Basic Expression Macro

Simple macro that wraps an expression in debugging output.

```slang
macro dbg {
    ($val:expr) => {
        {
            val __result = $val
            print(__result)
            __result
        }
    }
}

main = () {
    val x = dbg!(5 + 3)    // prints 8, x = 8
    print(x)               // prints 8
}
```

## Example 2: Multi-Argument Macro

Macro with multiple captures for variable swapping.

```slang
macro swap {
    ($a:ident, $b:ident) => {
        val __temp = $a
        $a = $b
        $b = __temp
    }
}

main = () {
    var x = 10
    var y = 20
    swap!(x, y)
    print(x)    // prints 20
    print(y)    // prints 10
}
```

## Example 3: Assertion Macro with Optional Message

Multi-rule macro demonstrating pattern matching for optional arguments.

```slang
macro assert {
    ($cond:expr) => {
        if !($cond) {
            print("Assertion failed")
            exit(1)
        }
    }
    ($cond:expr, $msg:expr) => {
        if !($cond) {
            print($msg)
            exit(1)
        }
    }
}

main = () {
    val x = 5
    assert!(x > 0)
    assert!(x < 10, "x should be less than 10")
    print("All assertions passed")
}
```

## Example 4: Variadic Macro with Repetition

Macro using repetition patterns for variable argument count.

```slang
macro print_all {
    ($($val:expr),+) => {
        $( print($val) )+
    }
}

main = () {
    print_all!(1, 2, 3)
    // Expands to:
    // print(1)
    // print(2)
    // print(3)
}
```

## Example 5: Max Macro with Multiple Rules

Demonstrates overloading by pattern matching on argument count.

```slang
macro max {
    ($a:expr, $b:expr) => {
        if $a > $b { $a } else { $b }
    }
    ($a:expr, $b:expr, $c:expr) => {
        max!(max!($a, $b), $c)
    }
}

main = () {
    val m1 = max!(3, 7)           // 7
    val m2 = max!(3, 7, 5)        // 7
    print(m1)
    print(m2)
}
```

## Example 6: Hygiene Demonstration

Shows that macro-introduced variables don't conflict with user variables.

```slang
macro with_temp {
    ($val:expr) => {
        {
            val temp = $val
            temp * 2
        }
    }
}

main = () {
    val temp = 100           // user's temp
    val result = with_temp!(5)   // macro's temp is renamed
    print(temp)              // prints 100 (unaffected)
    print(result)            // prints 10
}
```

## Example 7: Statement Macro

Macro that generates multiple statements.

```slang
macro repeat {
    ($n:expr, $body:stmt) => {
        {
            var __i = 0
            while __i < $n {
                $body
                __i = __i + 1
            }
        }
    }
}

main = () {
    repeat!(3, print("hello"))
    // prints "hello" three times
}
```

## Example 8: Compile-Time Error

Demonstrates error handling when pattern doesn't match.

```slang
macro need_two {
    ($a:expr, $b:expr) => {
        $a + $b
    }
}

main = () {
    // Error: macro pattern mismatch
    // 'need_two' expects exactly 2 expression arguments
    val x = need_two!(1, 2, 3)
}
```
