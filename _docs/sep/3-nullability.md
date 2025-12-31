# Status

DRAFT, 2025-12-26

# Summary/Motivation

Add Kotlin-style nullable types to Slang, enabling safe handling of potentially absent values through compile-time null safety. This introduces `T?` syntax for nullable types, safe call operator `?.` for chaining, and smart casts after null checks.

# Goals/Non-Goals

- [goal] Nullable type syntax with `?` suffix (e.g., `i64?`, `string?`, `User?`)
- [goal] Safe call operator `?.` for nullable field chaining
- [goal] Smart casts after null checks in `if` conditions
- [goal] `null` literal for nullable types
- [non-goal] Nullable primitive inference (primitives are non-null by default)
- [non-goal] `?.let {}` or `?.run {}` scope functions (can be added later)
- [non-goal] Elvis operator `?:` (may be added in future proposal)
- [non-goal] Non-null assertion operator `!!` (may be added in future proposal)
- [non-goal] Safe calls on methods (Slang doesn't have methods yet, only struct fields)

# Design Decisions

These decisions were made during implementation planning:

1. **Null Representation**: Use sentinel value `0x8000000000000000` (minimum i64, MSB set)
   - Chosen over tagged union (simpler codegen) and zero (ambiguous for i64)
   - Trade-off: This value cannot be used as a valid i64? value

2. **Smart Casts**: Include in first implementation
   - Flow-sensitive typing after null checks
   - Only for immutable `val` variables (mutable `var` could change)
   - Patterns: `if x != null { ... }` and early return `if x == null { return }`

3. **Safe Call Scope**: Struct fields only
   - `person?.address` where address is a nullable field
   - No method calls (Slang doesn't have methods on types yet)

# APIs

- `T?` - Nullable type modifier, indicating a value that may be `T` or `null`.
- `null` - Literal representing the absence of a value, assignable to any nullable type.
- `?.` - Safe call operator that returns `null` if receiver is `null`, otherwise accesses the field.

# Description

## Step 1: Lexer Changes

**File:** `compiler/lexer/lexer.go`

Add token support for nullable syntax:

```go
// New token types (add after line 77)
TokenTypeNull      // 'null' keyword
TokenTypeQuestion  // '?' for type syntax
TokenTypeSafeCall  // '?.' compound operator
```

Add to keywords map (line 27):
```go
"null": TokenTypeNull,
```

Tokenization logic (around line 522, after dot handling):
- On `?`: lookahead for `.` → emit `TokenTypeSafeCall`, else emit `TokenTypeQuestion`
- Pattern matches existing `==` vs `=` handling at lines 526-537

## Step 2: Parser Changes

**Files:** `compiler/ast/ast.go`, `compiler/parser/parser.go`

### AST Changes

Add to `LiteralType` enum:
```go
LiteralTypeNull  // null literal
```

Add new AST node:
```go
type SafeCallExpr struct {
    Object      Expression
    SafeCallPos Position
    Field       string
    FieldPos    Position
}
```

### Parser Changes

1. **Type parsing** (`parseTypeName()` at line 1406):
   - After parsing base type and generic parameters
   - Check for `TokenTypeQuestion`
   - Append `"?"` to type name string (e.g., `"i64?"`, `"User?"`)

2. **Null literal** (`ParseLiteral()` at line 795):
   - Add case for `TokenTypeNull`
   - Create `LiteralExpr{Kind: LiteralTypeNull, Value: "null"}`

3. **Safe call** (`parseExpression()` at line 869):
   - In postfix operator loop (where dot is handled)
   - Add case for `TokenTypeSafeCall`
   - Create `SafeCallExpr` node (same precedence as dot)

### Operator Precedence

```
.  ?.           (member access, safe call - highest)
! -             (unary prefix)
* / %           (multiplicative)
+ -             (additive)
< > <= >=       (comparison)
== !=           (equality)
&&              (logical and)
||              (logical or)
=               (assignment - lowest)
```

## Step 3: Type System Changes

**File:** `compiler/semantic/types.go`

Add nullable type wrapper (after line 291, following `ArrayType` pattern):

```go
// NullableType wraps a type to indicate it may be null
type NullableType struct {
    InnerType Type
}

func (t NullableType) String() string {
    return t.InnerType.String() + "?"
}

func (t NullableType) Equals(other Type) bool {
    o, ok := other.(NullableType)
    if !ok {
        return false
    }
    return t.InnerType.Equals(o.InnerType)
}
```

Add bottom type for null literal:
```go
// NothingType is the type of 'null', assignable to any T?
type NothingType struct{}

func (t NothingType) String() string { return "Nothing" }
func (t NothingType) Equals(other Type) bool {
    _, ok := other.(NothingType)
    return ok
}
```

Add helper functions:
```go
func IsNullable(t Type) bool {
    _, ok := t.(NullableType)
    return ok
}

func MakeNullable(t Type) Type {
    if IsNullable(t) {
        return t  // don't double-wrap
    }
    return NullableType{InnerType: t}
}

func UnwrapNullable(t Type) (Type, bool) {
    if n, ok := t.(NullableType); ok {
        return n.InnerType, true
    }
    return t, false
}
```

### Type Rules

| Expression | Input Type | Result Type |
|------------|------------|-------------|
| `x: T?` where `x = null` | `Nothing` | `T?` |
| `x: T?` where `x = value` | `T` | `T?` |
| `x?.field` | `T?` where T has field | `FieldType?` |

### Subtyping Rules

- `T` is subtype of `T?` (non-null can be assigned to nullable)
- `Nothing` is subtype of all `T?` (null assignable to any nullable)
- `T?` is NOT subtype of `T` (requires smart cast or explicit unwrap)

## Step 4: Semantic Analysis

**Files:** `compiler/semantic/typed_ast.go`, `compiler/semantic/analyzer.go`

### Typed AST

Add typed safe call expression:
```go
type TypedSafeCallExpr struct {
    Type        Type            // nullable result type
    Object      TypedExpression // nullable object
    SafeCallPos ast.Position
    Field       string
    FieldPos    ast.Position
    FieldOffset int             // byte offset for codegen
}
```

### Analyzer Changes

1. **Add smart cast tracking** to Analyzer struct (line 71):
   ```go
   smartCasts map[string]Type  // variable -> unwrapped type
   ```

2. **Update `resolveTypeName()`** (line 215):
   ```go
   // Check for T? syntax before other checks
   if strings.HasSuffix(name, "?") {
       innerName := name[:len(name)-1]
       innerType := a.resolveTypeName(innerName, pos)
       if _, isErr := innerType.(ErrorType); isErr {
           return TypeError
       }
       return NullableType{InnerType: innerType}
   }
   ```

3. **Handle null literal** in `analyzeLiteral()`:
   - Case `ast.LiteralTypeNull` → return `TypedLiteralExpr{Type: NothingType{}}`

4. **Add `analyzeSafeCallExpr()`**:
   - Analyze object expression
   - Verify object type is nullable (error if not: "safe call ?. on non-nullable type")
   - Unwrap to get inner struct type
   - Verify field exists on struct
   - Return `TypedSafeCallExpr` with nullable field type

5. **Update `checkTypeCompatibility()`** (line 510):
   ```go
   // null → T?: always allowed
   if _, isNothing := initType.(NothingType); isNothing {
       if IsNullable(declaredType) {
           return true
       }
       // Error: cannot assign null to non-nullable type
   }

   // T → T?: allowed (upcast)
   if !IsNullable(initType) && IsNullable(declaredType) {
       if nullable, ok := declaredType.(NullableType); ok {
           if nullable.InnerType.Equals(initType) {
               return true
           }
       }
   }

   // T? → T: error (requires smart cast)
   if IsNullable(initType) && !IsNullable(declaredType) {
       // Error: cannot assign T? to T, handle null first
   }
   ```

## Step 5: Smart Casts

**File:** `compiler/semantic/analyzer.go`

Implement flow-sensitive typing for null checks:

```slang
val x: i64? = getSomeValue()

if x != null {
    // x is automatically treated as i64 (smart cast)
    print(x + 10)  // OK - x is smart cast to i64
}

// x is still i64? here
print(x + 10)  // Error: cannot use i64? as i64
```

### Implementation

1. **In `analyzeIfStatement()`** (around line 983):
   ```go
   // Before analyzing then-branch, detect null check
   if varName, unwrappedType := a.detectNullCheck(stmt.Condition); varName != "" {
       // Save previous smart cast state
       oldCast, hadCast := a.smartCasts[varName]

       // Add smart cast for then-branch
       a.smartCasts[varName] = unwrappedType

       // Analyze then-branch with smart cast active
       thenTyped := a.analyzeBlockStatement(stmt.ThenBranch)

       // Restore previous state
       if hadCast {
           a.smartCasts[varName] = oldCast
       } else {
           delete(a.smartCasts, varName)
       }

       // Analyze else-branch without smart cast
       // ...
   }
   ```

2. **Implement `detectNullCheck()`**:
   ```go
   func (a *Analyzer) detectNullCheck(cond ast.Expression) (string, Type) {
       binExpr, ok := cond.(*ast.BinaryExpr)
       if !ok || binExpr.Op != "!=" {
           return "", nil
       }

       // Check for pattern: identifier != null
       ident, isIdent := binExpr.Left.(*ast.IdentifierExpr)
       lit, isLit := binExpr.Right.(*ast.LiteralExpr)
       if !isIdent || !isLit || lit.Kind != ast.LiteralTypeNull {
           return "", nil
       }

       // Check variable is immutable and nullable
       varInfo, exists := a.currentScope.Lookup(ident.Name)
       if !exists || varInfo.Mutable {
           return "", nil  // no smart cast for mutable vars
       }

       if inner, isNullable := UnwrapNullable(varInfo.Type); isNullable {
           return ident.Name, inner
       }
       return "", nil
   }
   ```

3. **Update `analyzeIdentifier()`** (around line 1664):
   ```go
   // Check for smart cast first
   if smartType, ok := a.smartCasts[name]; ok {
       return &TypedIdentifierExpr{
           Type: smartType,  // use smart-cast type
           Name: name,
           // ...
       }
   }
   // Fall through to normal lookup...
   ```

### Smart Cast Rules

- `if x != null { ... }` - x is non-null in the then branch
- `if x == null { return }` - x is non-null after the if (early return pattern)
- `x != null && x.field` - x is non-null on right side of `&&` (future enhancement)
- `x == null || ...` - x is non-null NOT guaranteed on right side
- Smart casts apply to `val` bindings only (mutable `var` could change)

## Step 6: Code Generation

**File:** `compiler/codegen/typed_codegen.go`

Generate ARM64 assembly for nullable operations.

### Null Representation

Use sentinel value `0x8000000000000000` (minimum i64):

```asm
// Load null sentinel into x2
mov x2, #0x8000
movk x2, #0x0, lsl #16
movk x2, #0x0, lsl #32
movk x2, #0x0, lsl #48
```

Helper function:
```go
func emitNullSentinel(builder *strings.Builder, reg string) {
    builder.WriteString(fmt.Sprintf("    mov %s, #0x8000\n", reg))
    builder.WriteString(fmt.Sprintf("    movk %s, #0x0, lsl #16\n", reg))
    builder.WriteString(fmt.Sprintf("    movk %s, #0x0, lsl #32\n", reg))
    builder.WriteString(fmt.Sprintf("    movk %s, #0x0, lsl #48\n", reg))
}
```

### Null Literal Generation

In `generateExpr()` for `TypedLiteralExpr` with null type:
```go
case NothingType:
    emitNullSentinel(&builder, "x2")
```

### Safe Call Generation (`x?.field`)

```asm
    // Evaluate object expression (result in x2)
    <object code>

    // Load null sentinel into x3 for comparison
    mov x3, #0x8000
    movk x3, #0x0, lsl #16
    movk x3, #0x0, lsl #32
    movk x3, #0x0, lsl #48

    // Check if object is null
    cmp x2, x3
    b.eq _safe_null_N

    // Object is not null, load field
    ldr x2, [x2, #<field_offset>]
    b _safe_done_N

_safe_null_N:
    mov x2, x3              // return null sentinel

_safe_done_N:
    // result in x2
```

Implementation:
```go
func (g *TypedCodeGenerator) generateSafeCallExpr(expr *TypedSafeCallExpr, ctx *BaseContext) (string, error) {
    builder := strings.Builder{}

    // Generate unique labels
    nullLabel := ctx.NextLabel("safe_null")
    doneLabel := ctx.NextLabel("safe_done")

    // Evaluate object
    objCode, err := g.generateExpr(expr.Object, ctx)
    if err != nil {
        return "", err
    }
    builder.WriteString(objCode)

    // Load null sentinel into x3
    emitNullSentinel(&builder, "x3")

    // Null check
    builder.WriteString("    cmp x2, x3\n")
    builder.WriteString(fmt.Sprintf("    b.eq %s\n", nullLabel))

    // Non-null path: load field
    builder.WriteString(fmt.Sprintf("    ldr x2, [x2, #%d]\n", expr.FieldOffset))
    builder.WriteString(fmt.Sprintf("    b %s\n", doneLabel))

    // Null path: return null
    builder.WriteString(fmt.Sprintf("%s:\n", nullLabel))
    builder.WriteString("    mov x2, x3\n")

    // Done
    builder.WriteString(fmt.Sprintf("%s:\n", doneLabel))

    return builder.String(), nil
}
```

### Null Comparison (`x == null`, `x != null`)

In binary expression handling:
```go
// Check if comparing against null
if isNullComparison(expr) {
    // Load value into x2
    valueCode, _ := g.generateExpr(getNonNullOperand(expr), ctx)
    builder.WriteString(valueCode)

    // Load null sentinel into x3
    emitNullSentinel(&builder, "x3")

    // Compare
    builder.WriteString("    cmp x2, x3\n")

    // Set result based on operator
    if expr.Op == "==" {
        builder.WriteString("    cset x2, eq\n")
    } else { // !=
        builder.WriteString("    cset x2, ne\n")
    }
}
```

## Error Handling

```slang
// Assigning null to non-nullable type
val x: i64 = null                 // Error: cannot assign null to non-nullable type i64

// Using nullable where non-nullable expected
val x: i64? = 42
val y: i64 = x                    // Error: cannot assign i64? to i64, handle null first

// Safe call on non-nullable
val x: i64 = 42
print(x?.toString())              // Error: safe call ?. on non-nullable type i64

// Chaining without safe call
val street = person.address.street  // Error if address is nullable: use ?. for nullable access
```

# Implementation Order

1. **Lexer** - Add `null`, `?`, `?.` tokens
2. **AST** - Add `LiteralTypeNull`, `SafeCallExpr`
3. **Parser** - Parse `T?` types, `null` literal, `?.` safe calls
4. **Types** - Add `NullableType`, `NothingType`, helpers
5. **Typed AST** - Add `TypedSafeCallExpr`
6. **Semantic** - Type checking, safe call analysis
7. **Smart Casts** - Flow-sensitive typing
8. **Codegen** - Null sentinel, safe call branching
9. **E2E Tests** - Integration tests

# Files to Modify

| File | Changes |
|------|---------|
| `compiler/lexer/lexer.go` | Add `TokenTypeNull`, `TokenTypeQuestion`, `TokenTypeSafeCall` |
| `compiler/ast/ast.go` | Add `LiteralTypeNull`, `SafeCallExpr` |
| `compiler/parser/parser.go` | Parse `T?`, `null`, `?.` |
| `compiler/semantic/types.go` | Add `NullableType`, `NothingType`, helpers |
| `compiler/semantic/typed_ast.go` | Add `TypedSafeCallExpr` |
| `compiler/semantic/analyzer.go` | Type checking, smart casts |
| `compiler/codegen/typed_codegen.go` | Null sentinel, safe call codegen |

# Risks and Limitations

1. **Sentinel Value Conflict**: `0x8000000000000000` cannot be used as a valid `i64?` value. This is an acceptable trade-off for simpler codegen.

2. **Smart Cast Soundness**: Smart casts only apply to immutable `val` variables. Mutable `var` could be reassigned between check and use.

3. **Safe Call Performance**: Each `?.` introduces a branch. For hot paths, explicit null checks may be more efficient.

4. **No Method Calls**: Safe calls only work with struct fields, not methods (Slang doesn't have methods yet).

# Alternatives

1. **Option/Maybe type (`Option<T>`)**: More explicit but verbose. Would require `Some(value)` and `None` everywhere. Kotlin-style `?` is more ergonomic for a systems language.

2. **Implicit null checks (like Go)**: Go allows nil checks but doesn't enforce them. This leads to runtime panics. Compile-time null safety is safer.

3. **Non-nullable by default with explicit `!` for non-null (like Dart)**: Dart uses `Type?` for nullable and `variable!` for assertion. We could add `!!` later if needed.

4. **Result types only**: Could use `Result<T, Error>` instead of nullability. Better for error handling but overkill for simple "might not exist" cases. Both can coexist.

5. **C++-style `std::optional`**: Requires explicit `.value()` or `.value_or()` calls. Less ergonomic than operators like `?.`.

6. **Tagged Union**: 8-byte value + 1-byte null flag. Clearer semantics but 2x memory usage and more complex codegen.

# Testing

- **Lexer tests**: Token recognition for `null`, `?`, `?.`
- **Parser tests**:
  - Nullable type parsing (`i64?`, `string?`, `User?`)
  - Safe call expressions (`x?.field`)
  - Chained expressions (`a?.b?.c`)
- **Semantic tests**:
  - Type checking nullable assignments
  - Type inference for nullable expressions
  - Smart cast in if conditions
  - Error detection for null safety violations
- **Codegen tests**:
  - Null representation in memory
  - Safe call branching logic
- **E2E tests** in `_examples/slang/nullability/`:
  - Basic nullable assignment and null checks
  - Safe call chaining
  - Smart cast patterns
  - Integration with structs

# Code Examples

## Example 1: Basic Nullable Types

Demonstrates declaring nullable variables and basic null checks.

```slang
main = () {
    val x: i64? = 42              // nullable with value
    val y: i64? = null            // nullable with null

    if x != null {
        print(x)                  // prints: 42 (smart cast to i64)
    }

    if y != null {
        print(y)                  // not executed
    } else {
        print("y is null")        // prints: y is null
    }
}
```

## Example 2: Safe Call Operator

Shows chaining through nullable values without explicit null checks.

```slang
Address = struct {
    val street: string
    val city: string
}

Person = struct {
    val name: string
    val address: Address?
}

main = () {
    val person1 = Person{ "Alice", Address{ "123 Main St", "Springfield" } }
    val person2 = Person{ "Bob", null }

    // Safe call chains - returns null if any part is null
    val street1: string? = person1.address?.street
    val street2: string? = person2.address?.street

    if street1 != null {
        print(street1)            // prints: 123 Main St
    }

    if street2 != null {
        print(street2)            // not executed
    } else {
        print("unknown")          // prints: unknown
    }
}
```

## Example 3: Smart Casts

Demonstrates automatic type narrowing after null checks.

```slang
processValue = (x: i64?) -> i64 {
    // Early return pattern - x is smart cast after this
    if x == null {
        return 0
    }

    // x is now i64 (smart cast), not i64?
    x * 2
}

main = () {
    print(processValue(21))       // prints: 42
    print(processValue(null))     // prints: 0
}
```

## Example 4: Complex Chaining

Demonstrates complex nullable chains with safe calls.

```slang
Company = struct {
    val name: string
    val ceo: Person?
}

Person = struct {
    val name: string
    val assistant: Person?
}

getCompany = (id: i64) -> Company? {
    // ... lookup logic
}

main = () {
    val company = getCompany(1)

    // Deep nullable chain
    val assistantName: string? = company?.ceo?.assistant?.name
    if assistantName != null {
        print(assistantName)
    } else {
        print("No assistant")
    }

    // Safe alternative with explicit checks
    if company != null {
        val ceo = company.ceo
        if ceo != null {
            print(ceo.name)
        } else {
            print("Unknown CEO")
        }
    }
}
```

## Example 5: Nullable in Collections

Shows nullable element types in arrays.

```slang
main = () {
    val maybeNumbers: Array<i64?> = [1, null, 3, null, 5]

    var sum = 0
    for item in maybeNumbers {
        if item != null {
            sum = sum + item      // smart cast to i64
        }
    }
    print(sum)                    // prints: 9
}
```

## Example 6: Returning Nullable Values

Shows functions that may return null.

```slang
User = struct {
    val name: string
    val id: i64
}

findUser = (id: i64) -> User? {
    if id == 1 {
        User{ "admin", 1 }
    } else {
        null
    }
}

main = () {
    val user1 = findUser(1)
    val user2 = findUser(999)

    if user1 != null {
        print(user1.name)         // prints: admin
    }

    if user2 != null {
        print(user2.name)         // not executed
    } else {
        print("User not found")   // prints: User not found
    }
}
```

## Example 7: Combining Safe Calls with Conditionals

Shows practical patterns for working with nullable values.

```slang
Config = struct {
    val timeout: i64?
    val retries: i64?
}

loadConfig = () -> Config? {
    // ... load from file
}

main = () {
    val config = loadConfig()

    // Pattern: check and use
    val timeout = config?.timeout
    if timeout != null {
        print("Timeout: ")
        print(timeout)
    } else {
        print("Using default timeout")
    }

    // Pattern: nested checks with smart casts
    if config != null {
        // config is non-null here
        if config.retries != null {
            print("Retries: ")
            print(config.retries)  // smart cast to i64
        }
    }
}
```

# Future Enhancements

These are explicitly out of scope but may be added later:

1. **Elvis operator `?:`**
   ```slang
   val x: i64 = maybeValue ?: defaultValue
   ```

2. **Non-null assertion `!!`**
   ```slang
   val x: i64 = nullableValue!!  // panic if null
   ```

3. **Scope functions `.let`, `.also`**
   ```slang
   value?.let { v -> print(v) }
   ```

4. **Smart cast in `&&` expressions**
   ```slang
   if x != null && x > 10 { ... }  // x smart cast on right side
   ```
