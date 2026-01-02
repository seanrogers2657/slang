# Status

IMPLEMENTED, 2026-01-01

# Summary/Motivation

Add Kotlin-style nullable types to Slang, enabling safe handling of potentially absent values through compile-time null safety. This introduces `T?` syntax for nullable types and safe call operator `?.` for chaining.

# Goals/Non-Goals

- [implemented] Nullable type syntax with `?` suffix (e.g., `i64?`, `string?`, `User?`)
- [implemented] Safe call operator `?.` for nullable field chaining
- [implemented] `null` literal for nullable types
- [future] Smart casts after null checks in `if` conditions
- [non-goal] Nullable primitive inference (primitives are non-null by default)
- [non-goal] `?.let {}` or `?.run {}` scope functions (can be added later)
- [non-goal] Elvis operator `?:` (may be added in future proposal)
- [non-goal] Non-null assertion operator `!!` (may be added in future proposal)
- [non-goal] Safe calls on methods (Slang doesn't have methods yet, only struct fields)

# Design Decisions

These decisions were made during implementation planning:

1. **Null Representation**: Type-dependent
   - **Primitives (`i64?`, `bool?`)**: Tagged union (8-byte tag + 8-byte value = 16 bytes)
   - **Reference types (`Struct?`, `string?`)**: Nullable pointer (8 bytes, null = 0)
   - This optimizes memory for struct nullables while preserving full primitive range

2. **Smart Casts**: Deferred to future implementation
   - Flow-sensitive typing after null checks
   - Only for immutable `val` variables (mutable `var` could change)
   - Patterns: `if x != null { ... }` and early return `if x == null { return }`
   - Smart cast applies in `&&` expressions: `if x != null && x > 10` works

3. **Safe Call Scope**: Struct fields only
   - `person?.address` where address is a nullable field
   - No method calls (Slang doesn't have methods on types yet)

4. **Nested Nullables**: Disallowed
   - `T??` is a compile error: "nested nullable types are not allowed"
   - Rationale: No practical use case, adds complexity

5. **Type Inference with `null`**: Requires explicit type
   - `val x = null` is a compile error: "cannot infer type from null, add type annotation"
   - `val x: i64? = null` is valid

6. **Null Equality**: `null == null` is `true`
   - Comparing two nullable values: both null → true, one null → false, both non-null → compare values

7. **Arithmetic on Nullables**: Compile error
   - `i64?` and `i64` are distinct types with no implicit downcasting
   - `x + 10` where x is `i64?` → error: "cannot use i64? as i64, handle null first"
   - Must smart cast or explicitly check null before arithmetic

8. **Function Parameter Passing**: By value
   - 16-byte tagged nullables are copied when passed to functions
   - Consistent with primitive pass-by-value semantics

9. **Default Initialization**: Implicitly null
   - `var x: i64?` without initializer defaults to null
   - `val x: i64?` without initializer is an error (immutable must be initialized)

10. **Printing Nullables**: Prints "null" string
    - `print(x)` where x is null outputs the literal string "null"

11. **Implicit Upcasting**: `T` → `T?` is automatic
    - Passing `i64` to parameter expecting `i64?` auto-wraps
    - Assigning `i64` to `var x: i64?` auto-wraps (type remains `i64?`)
    - Reverse (`T?` → `T`) requires smart cast or explicit null check

12. **Function Return Convention**: Two registers for nullable primitives
    - `x0` = tag (0 = null, 1 = has value)
    - `x1` = value (when tag = 1)
    - Nullable references return in single register (pointer or 0)

13. **Safe Call Chain Type**: Result is always `InnerType?`
    - `a?.b?.c` where `c: T` results in `T?`
    - Nullability propagates but doesn't nest (no `T??`)

14. **Multiple Smart Casts in `&&`**: All null checks apply
    - `if x != null && y != null { ... }` smart casts both x and y
    - Each `!= null` check adds to the smart cast set

15. **Null Comparison Result**: Non-nullable `bool`
    - `x == null` and `x != null` return `bool`, not `bool?`
    - These are compile-time known to always produce a boolean

16. **Loop Smart Casts**: Disabled for safety
    - `while x != null { ... }` does NOT smart cast x in the body
    - Loop body could reassign x, making smart cast unsound
    - Use explicit local: `val y = x; if y != null { ... }`

17. **If-Else Type Inference**: Union of branch types
    - `if cond { 42 } else { null }` infers `i64?`
    - If one branch is `T` and another is `null`, result is `T?`

18. **Struct Field Layout**: Fields sized by their types
    - `val x: i64` = 8 bytes
    - `val y: i64?` = 16 bytes (tagged union)
    - `val p: Point?` = 8 bytes (nullable pointer)
    - Struct total size = sum of field sizes (with alignment padding)

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
- `x != null && x.field` - x is non-null on right side of `&&`
- `x != null && x > 10` - x is smart cast for entire right-hand side of `&&`
- `x == null || ...` - x is non-null NOT guaranteed on right side (short-circuit means x could be null)
- Smart casts apply to `val` bindings only (mutable `var` could change)

### Smart Cast in `&&` Expressions

When analyzing `&&` expressions, apply smart casts discovered on the left side to the right side. This works recursively for chained `&&`:

```go
// detectNullChecks returns all null checks in an expression
// Handles: x != null, x != null && y != null, etc.
func (a *Analyzer) detectNullChecks(expr ast.Expression) map[string]Type {
    casts := make(map[string]Type)

    switch e := expr.(type) {
    case *ast.BinaryExpr:
        if e.Op == "!=" {
            // Check for pattern: identifier != null
            if ident, ok := e.Left.(*ast.IdentifierExpr); ok {
                if lit, ok := e.Right.(*ast.LiteralExpr); ok && lit.Kind == ast.LiteralTypeNull {
                    varInfo, exists := a.currentScope.Lookup(ident.Name)
                    if exists && !varInfo.Mutable {
                        if inner, isNullable := UnwrapNullable(varInfo.Type); isNullable {
                            casts[ident.Name] = inner
                        }
                    }
                }
            }
        } else if e.Op == "&&" {
            // Recursively collect from both sides
            for k, v := range a.detectNullChecks(e.Left) {
                casts[k] = v
            }
            for k, v := range a.detectNullChecks(e.Right) {
                casts[k] = v
            }
        }
    }
    return casts
}

func (a *Analyzer) analyzeLogicalAnd(expr *ast.BinaryExpr) TypedExpression {
    // Analyze left side first
    leftTyped := a.analyzeExpr(expr.Left)

    // Collect all null checks from left side
    newCasts := a.detectNullChecks(expr.Left)

    // Save old casts and apply new ones
    oldCasts := make(map[string]Type)
    for name, typ := range newCasts {
        if old, exists := a.smartCasts[name]; exists {
            oldCasts[name] = old
        }
        a.smartCasts[name] = typ
    }

    // Analyze right side with smart casts active
    rightTyped := a.analyzeExpr(expr.Right)

    // Restore old state
    for name := range newCasts {
        if old, existed := oldCasts[name]; existed {
            a.smartCasts[name] = old
        } else {
            delete(a.smartCasts, name)
        }
    }

    return &TypedBinaryExpr{Left: leftTyped, Op: "&&", Right: rightTyped, ...}
}
```

Example:
```slang
if x != null && y != null && x + y > 10 {
    // Both x and y are smart cast to non-nullable here
    print(x + y)
}
```

## Step 6: Code Generation

**File:** `compiler/codegen/typed_codegen.go`

Generate ARM64 assembly for nullable operations.

### Null Representation

**Type-dependent representation:**

#### Primitives (`i64?`, `bool?`) - Tagged Union (16 bytes)

```
┌─────────────────┬─────────────────┐
│ tag (8B aligned)│ value (8B)      │
└─────────────────┴─────────────────┘
Offset 0: tag (0 = null, 1 = has value)
Offset 8: actual value (when tag = 1)
```

Note: Tag uses 8 bytes (not 1) due to ARM64 alignment requirements.

#### Reference Types (`Struct?`, `string?`) - Nullable Pointer (8 bytes)

```
┌─────────────────┐
│ pointer (8B)    │  0 = null, non-zero = valid address
└─────────────────┘
```

Structs are already references, so `Struct?` is simply a pointer that can be 0.

### Helper: Determine Nullable Kind

```go
func isReferenceType(t Type) bool {
    switch t.(type) {
    case StructType, StringType:
        return true
    default:
        return false
    }
}

func nullableSize(inner Type) int {
    if isReferenceType(inner) {
        return 8   // nullable pointer
    }
    return 16      // tagged union
}
```

### Helper Functions

```go
// emitNullCheck checks if nullable is null, branches to label if so
// For primitives: checks tag byte at offset 0
// For references: checks if pointer is 0
func emitNullCheck(builder *strings.Builder, reg string, isReference bool, nullLabel string) {
    if isReference {
        // Reference type: just check if pointer is 0
        builder.WriteString(fmt.Sprintf("    cbz %s, %s\n", reg, nullLabel))
    } else {
        // Primitive: check tag byte
        builder.WriteString(fmt.Sprintf("    ldrb w3, [%s]\n", reg))
        builder.WriteString(fmt.Sprintf("    cbz w3, %s\n", nullLabel))
    }
}

// emitLoadNullableValue loads the value from a nullable (assumes not null)
// For primitives: loads from offset 8
// For references: value is the pointer itself
func emitLoadNullableValue(builder *strings.Builder, src, dest string, isReference bool) {
    if isReference {
        // Reference: the pointer IS the value
        if src != dest {
            builder.WriteString(fmt.Sprintf("    mov %s, %s\n", dest, src))
        }
    } else {
        // Primitive: load from offset 8
        builder.WriteString(fmt.Sprintf("    ldr %s, [%s, #8]\n", dest, src))
    }
}

// emitStoreNull stores null to a nullable location
func emitStoreNull(builder *strings.Builder, dest string, isReference bool) {
    if isReference {
        // Reference: store null pointer
        builder.WriteString(fmt.Sprintf("    str xzr, [%s]\n", dest))
    } else {
        // Primitive: set tag = 0
        builder.WriteString(fmt.Sprintf("    str xzr, [%s]\n", dest))
    }
}

// emitStoreNullableValue stores a value to a nullable location
func emitStoreNullableValue(builder *strings.Builder, dest, value string, isReference bool) {
    if isReference {
        // Reference: store the pointer directly
        builder.WriteString(fmt.Sprintf("    str %s, [%s]\n", value, dest))
    } else {
        // Primitive: set tag = 1, store value at offset 8
        builder.WriteString(fmt.Sprintf("    mov x3, #1\n"))
        builder.WriteString(fmt.Sprintf("    str x3, [%s]\n", dest))
        builder.WriteString(fmt.Sprintf("    str %s, [%s, #8]\n", value, dest))
    }
}

// allocateNullable allocates stack space for a nullable
func (ctx *CodeGenContext) allocateNullable(inner Type) int {
    return ctx.AllocateStack(nullableSize(inner))
}
```

### Null Literal Generation

In `generateExpr()` for `TypedLiteralExpr` with null type:
```go
case NothingType:
    // For null literal, we need to create a tagged null value
    // If assigning to a variable, the variable location will have tag=0
    // For intermediate expressions, we use a convention where x2=0 indicates null
    builder.WriteString("    mov x2, #0\n")  // tag indicating null
```

### Variable Storage for Nullables

When declaring a nullable variable:
```go
func (g *TypedCodeGenerator) generateNullableVarDecl(decl *TypedVarDecl, ctx *CodeGenContext) (string, error) {
    builder := strings.Builder{}

    // Allocate 16 bytes on stack
    offset := ctx.AllocateStack(16)
    ctx.SetVarOffset(decl.Name, offset)

    if isNullLiteral(decl.Init) {
        // Store null: just set tag to 0
        builder.WriteString(fmt.Sprintf("    str xzr, [x29, #%d]\n", -offset))
    } else {
        // Evaluate expression
        initCode, _ := g.generateExpr(decl.Init, ctx)
        builder.WriteString(initCode)

        // Store with tag = 1
        builder.WriteString(fmt.Sprintf("    mov x3, #1\n"))
        builder.WriteString(fmt.Sprintf("    str x3, [x29, #%d]\n", -offset))
        builder.WriteString(fmt.Sprintf("    str x2, [x29, #%d]\n", -offset-8))
    }

    return builder.String(), nil
}
```

### Safe Call Generation (`x?.field`)

```asm
    // x2 contains address of nullable object
    <evaluate object address into x2>

    // Check if object is null (tag at offset 0)
    ldrb w3, [x2]
    cbz w3, _safe_null_N

    // Object is not null, load object pointer then field
    ldr x2, [x2, #8]                    // load actual object address
    ldr x2, [x2, #<field_offset>]       // load field from object
    b _safe_done_N

_safe_null_N:
    // Return null - set tag to 0 in result
    mov x2, #0

_safe_done_N:
    // x2 contains result (0 if null, field value otherwise)
```

Implementation:
```go
func (g *TypedCodeGenerator) generateSafeCallExpr(expr *TypedSafeCallExpr, ctx *BaseContext) (string, error) {
    builder := strings.Builder{}

    // Generate unique labels
    nullLabel := ctx.NextLabel("safe_null")
    doneLabel := ctx.NextLabel("safe_done")

    // Evaluate object (gets address of nullable)
    objCode, err := g.generateExpr(expr.Object, ctx)
    if err != nil {
        return "", err
    }
    builder.WriteString(objCode)

    // Null check via tag byte
    emitNullCheck(&builder, "x2", nullLabel)

    // Non-null path: load value then field
    emitLoadNullableValue(&builder, "x2", "x2")
    builder.WriteString(fmt.Sprintf("    ldr x2, [x2, #%d]\n", expr.FieldOffset))
    builder.WriteString(fmt.Sprintf("    b %s\n", doneLabel))

    // Null path: return null indicator
    builder.WriteString(fmt.Sprintf("%s:\n", nullLabel))
    builder.WriteString("    mov x2, #0\n")

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
    // Get address of nullable value
    valueCode, _ := g.generateExpr(getNonNullOperand(expr), ctx)
    builder.WriteString(valueCode)

    // Load tag byte and compare to 0
    builder.WriteString("    ldrb w3, [x2]\n")

    // Set result based on operator
    if expr.Op == "==" {
        // x == null: true if tag == 0
        builder.WriteString("    cmp w3, #0\n")
        builder.WriteString("    cset x2, eq\n")
    } else { // !=
        // x != null: true if tag != 0
        builder.WriteString("    cmp w3, #0\n")
        builder.WriteString("    cset x2, ne\n")
    }
}
```

### Function Return Convention

For functions returning nullable primitives, use two registers:

```asm
// Returning null from -> i64?
mov x0, #0              // tag = 0 (null)
// x1 undefined
ret

// Returning 42 from -> i64?
mov x0, #1              // tag = 1 (has value)
mov x1, #42             // value
ret
```

For functions returning nullable references (`-> Struct?`):
```asm
// Returning null
mov x0, #0              // null pointer
ret

// Returning struct pointer
mov x0, x2              // pointer to struct
ret
```

Caller code for handling nullable primitive return:
```asm
    bl maybeGetValue        // call function
    cbz x0, _was_null       // check tag
    // x1 contains the value
    mov x2, x1              // use value
    b _done
_was_null:
    // handle null case
_done:
```

### Stack Layout Example

For a function with nullable variables:
```slang
Point = struct { val x: i64, val y: i64 }

foo = () {
    val a: i64? = 42           // primitive nullable (16 bytes)
    val b: i64? = null         // primitive nullable (16 bytes)
    val c: i64 = 10            // non-nullable (8 bytes)
    val p: Point? = null       // struct nullable (8 bytes - just a pointer)
    val q: Point? = Point{1,2} // struct nullable (8 bytes - pointer to struct)
}
```

Stack layout:
```
        ┌─────────────────┐
x29-8   │ a.tag (8B)      │  = 1
x29-16  │ a.value (8B)    │  = 42
        ├─────────────────┤
x29-24  │ b.tag (8B)      │  = 0 (null)
x29-32  │ b.value (8B)    │  = (undefined)
        ├─────────────────┤
x29-40  │ c (8B)          │  = 10
        ├─────────────────┤
x29-48  │ p (8B)          │  = 0 (null pointer)
        ├─────────────────┤
x29-56  │ q (8B)          │  = <address of Point{1,2}>
        └─────────────────┘
```

Note: `p` and `q` are just 8-byte pointers since `Point` is a reference type.

## Error Handling

```slang
// Assigning null to non-nullable type
val x: i64 = null                 // Error: cannot assign null to non-nullable type 'i64'

// Using nullable where non-nullable expected
val x: i64? = 42
val y: i64 = x                    // Error: cannot assign 'i64?' to 'i64', handle null first

// Arithmetic on nullable
val x: i64? = 42
val y = x + 10                    // Error: cannot use 'i64?' as 'i64', handle null first

// Safe call on non-nullable
val x: i64 = 42
print(x?.toString())              // Error: safe call '?.' used on non-nullable type 'i64'

// Chaining without safe call
val street = person.address.street  // Error if address is nullable: use '?.' for nullable access

// Nested nullable
val x: i64?? = null               // Error: nested nullable types are not allowed

// Bare null without type annotation
val x = null                      // Error: cannot infer type from null, add type annotation

// Mutable smart cast attempt
var x: i64? = 42
if x != null {
    x = null                      // allowed (x is mutable)
    print(x + 1)                  // Error: 'x' was reassigned and smart cast no longer applies
}
```

# Implementation Order

1. ✅ **Lexer** - Add `null`, `?`, `?.` tokens
2. ✅ **AST** - Add `LiteralTypeNull`, `SafeCallExpr`
3. ✅ **Parser** - Parse `T?` types, `null` literal, `?.` safe calls
4. ✅ **Types** - Add `NullableType`, `NothingType`, helpers
5. ✅ **Typed AST** - Add `TypedSafeCallExpr`
6. ✅ **Semantic** - Type checking, safe call analysis
7. ⏳ **Smart Casts** - Flow-sensitive typing (deferred)
8. ✅ **Codegen** - Tagged union layout, safe call branching
9. ✅ **E2E Tests** - Integration tests

# Files Modified

| File | Changes | Status |
|------|---------|--------|
| `compiler/lexer/lexer.go` | Add `TokenTypeNull`, `TokenTypeQuestion`, `TokenTypeSafeCall` | ✅ |
| `compiler/ast/ast.go` | Add `LiteralTypeNull`, `SafeCallExpr` | ✅ |
| `compiler/parser/parser.go` | Parse `T?`, `null`, `?.` | ✅ |
| `compiler/semantic/types.go` | Add `NullableType`, `NothingType`, helpers | ✅ |
| `compiler/semantic/typed_ast.go` | Add `TypedSafeCallExpr` | ✅ |
| `compiler/semantic/analyzer.go` | Type checking, safe call analysis | ✅ |
| `compiler/codegen/typed_codegen.go` | Tagged union layout, safe call codegen | ✅ |

# Risks and Limitations

1. **Memory Overhead for Primitives**: Tagged union uses 16 bytes per nullable primitive (vs 8 bytes for non-nullable). This is the trade-off for preserving the full value range. Struct nullables remain 8 bytes.

2. **Smart Cast Soundness** (future): When smart casts are implemented, they will only apply to immutable `val` variables. Mutable `var` could be reassigned between check and use.

3. **Safe Call Performance**: Each `?.` introduces a branch. For hot paths, explicit null checks may be more efficient.

4. **No Method Calls**: Safe calls only work with struct fields, not methods (Slang doesn't have methods yet).

5. **Alignment Padding**: Tag byte in primitive nullables is padded to 8 bytes for ARM64 alignment, wasting 7 bytes. Could optimize with packed layouts in future.

6. **Two Null Representations**: Primitives use tagged union while references use null pointers. Codegen must handle both, adding complexity.

# Alternatives Considered

1. **Sentinel Value (`0x8000000000000000`)**: Uses minimum i64 as null marker. Simpler codegen (8 bytes, no tag) but "steals" one valid integer value. Rejected because losing a valid value is a footgun for a safety-focused language.

2. **Option/Maybe type (`Option<T>`)**: More explicit but verbose. Would require `Some(value)` and `None` everywhere. Kotlin-style `?` is more ergonomic for a systems language.

3. **Implicit null checks (like Go)**: Go allows nil checks but doesn't enforce them. This leads to runtime panics. Compile-time null safety is safer.

4. **Non-nullable by default with explicit `!` for non-null (like Dart)**: Dart uses `Type?` for nullable and `variable!` for assertion. We could add `!!` later if needed.

5. **Result types only**: Could use `Result<T, Error>` instead of nullability. Better for error handling but overkill for simple "might not exist" cases. Both can coexist.

6. **C++-style `std::optional`**: Requires explicit `.value()` or `.value_or()` calls. Less ergonomic than operators like `?.`.

7. **Pointer Boxing**: Nullable primitives become heap-allocated pointers. Natural null representation (address 0) but requires allocation and indirection for every access.

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
  - Tagged union layout (16-byte nullable storage)
  - Safe call branching logic
  - Null comparison via tag byte
- **E2E tests** in `_examples/slang/nullability/`:
  - Basic nullable assignment and null checks
  - Safe call chaining (implemented but needs E2E tests)
  - Integration with structs
  - Smart cast patterns (not yet implemented)

# Code Examples

## Example 1: Basic Nullable Types

Demonstrates declaring nullable variables and basic null checks.

```slang
main = () {
    val x: i64? = 42              // nullable with value
    val y: i64? = null            // nullable with null

    print(x != null)              // prints: true
    print(y == null)              // prints: true
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

## Example 3: Smart Casts (Future)

**Note:** Smart casts are not yet implemented. This example shows the intended future behavior.

```slang
processValue = (x: i64?) -> i64 {
    // Early return pattern - x would be smart cast after this
    if x == null {
        return 0
    }

    // x would be i64 (smart cast), not i64?
    x * 2
}

main = () {
    print(processValue(21))       // would print: 42
    print(processValue(null))     // would print: 0
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

1. **Smart casts after null checks**
   ```slang
   val x: i64? = getSomeValue()
   if x != null {
       print(x + 10)  // x would be smart cast to i64
   }
   ```

2. **Elvis operator `?:`**
   ```slang
   val x: i64 = maybeValue ?: defaultValue
   ```

3. **Non-null assertion `!!`**
   ```slang
   val x: i64 = nullableValue!!  // panic if null
   ```

4. **Scope functions `.let`, `.also`**
   ```slang
   value?.let { v -> print(v) }
   ```
