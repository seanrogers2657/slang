# Status

IMPLEMENTED, 2026-01-22

## Implementation Status

| Feature | Status | Notes |
|---------|--------|-------|
| `class` keyword | ✅ Implemented | |
| `self` keyword | ✅ Implemented | |
| `object` keyword | ✅ Implemented | |
| `?:` Elvis operator | ✅ Implemented | |
| ClassDecl AST node | ✅ Implemented | |
| ObjectDecl AST node | ✅ Implemented | |
| MethodDecl AST node | ✅ Implemented | |
| MethodCallExpr AST node | ✅ Implemented | |
| SelfExpr AST node | ✅ Implemented | |
| ClassType semantic type | ✅ Implemented | |
| ObjectType semantic type | ✅ Implemented | |
| TypedMethodCallExpr | ✅ Implemented | Unified typed AST for both instance and static method calls |
| Instance method codegen | ✅ Implemented | |
| Static method codegen | ✅ Implemented | |
| Safe navigation (?.) | ✅ Implemented | |
| Method overloading | ✅ Implemented | |
| E2E tests | ✅ Implemented | Test files in `_examples/slang/classes/` |

**Prerequisite:** SEP 1 (Pointers and Memory) - ✅ IMPLEMENTED

# Summary/Motivation

Add classes to Slang, extending structs with methods (functions bound to a type). This enables encapsulation of data and behavior together, allowing types to have associated operations without passing the instance explicitly to every function.

# Goals/Non-Goals

- [goal] Class declaration syntax with `class` keyword using assignment-based format
- [goal] Instance methods: have `self` as first parameter, called on instances
- [goal] Static methods: no `self` parameter, called on class name
- [goal] `self` keyword for accessing the current instance within methods
- [goal] Method calls using dot notation (`instance.method()` or `ClassName.staticMethod()`)
- [goal] Field declarations with `val`/`var` (like structs)
- [goal] Direct field construction via struct-literal syntax (`ClassName{ ... }`)
- [goal] Singleton objects via `object` keyword for static-only utility containers
- [goal] Method overloading (multiple methods with same name but different signatures)

## Ownership Integration (SEP 1)
- [goal] Method receiver types: `self: &T` (immutable borrow), `self: &&T` (mutable borrow), `self: *T` (takes ownership)
- [goal] Static factory methods returning `*ClassName` for heap allocation
- [goal] Class fields can be owned pointers (`*T`)
- [goal] Class instances work with `&T` and `*T` parameter types

## Non-Goals
- [non-goal] Inheritance (single or multiple)
- [non-goal] Visibility modifiers (`public`, `private`, `protected`)
- [non-goal] Abstract classes or interfaces
- [non-goal] Operator overloading
- [non-goal] Generic/parameterized classes (future enhancement)
- [non-goal] Properties with custom getters/setters
- [non-goal] Constructor overloading (use static factory methods instead)
- [non-goal] Default field values (use static factory methods for defaults)

# APIs

- `class` - Keyword for declaring a class type with fields and methods.
- `self` - Keyword referencing the current instance within method bodies. Also used as first parameter name for instance methods.
- `.method()` - Dot notation for calling methods on instances.
- `ClassName.method()` - Calling static methods on the class name.
- `ClassName{ ... }` - Direct field construction (same as struct syntax). Supports both positional and named arguments:
  ```slang
  val p1 = Point{ 10, 20 }           // positional: fields in declaration order
  val p2 = Point{ y: 20, x: 10 }     // named: any order, more explicit
  ```
- `object` - Keyword for declaring a singleton with static methods only (cannot be instantiated).
- `?.method()` - Safe navigation for method calls. Returns nullable type of the method's return type. If receiver is null, returns null (or no-op for void methods).
- `?:` - Elvis operator (null coalescing). Returns left side if non-null, otherwise right side. Precedence is lower than arithmetic operators (follows Kotlin).

## Singleton Objects

For utility containers with only static methods, use `object` instead of `class`:

```slang
Math = object {
    max = (a: i64, b: i64) -> i64 {
        when {
            a > b -> a
            else -> b
        }
    }

    min = (a: i64, b: i64) -> i64 {
        when {
            a < b -> a
            else -> b
        }
    }
}

main = () {
    print(Math.max(10, 20))  // prints: 20
    // val m = Math{}        // Error: cannot instantiate object
}
```

**Object vs Class:**

| Feature | `class` | `object` |
|---------|---------|----------|
| Fields | Yes | No |
| Instance methods | Yes | No |
| Static methods | Yes | Yes |
| Can instantiate | Yes | No |
| Use case | Data + behavior | Utility functions, namespacing |

Note: A class with no fields can still be instantiated (`Utils{}`), producing a zero-size instance. This is valid but typically pointless - prefer `object` for static-only containers.

## Methods on Temporaries

Method calls on temporary (literal) values are valid:

```slang
Point = class {
    val x: i64
    val y: i64

    magnitude = (self: &Point) -> i64 {
        self.x * self.x + self.y * self.y
    }
}

main = () {
    // Method call on temporary - valid
    val mag = Point{ 3, 4 }.magnitude()
    print(mag)  // prints: 25

    // Chained construction and method call
    print(Point{ 0, 0 }.magnitude())  // prints: 0
}
```

The temporary exists for the duration of the expression and is automatically borrowed for the method call.

## Safe Navigation with Method Calls

The safe navigation operator `?.` can be used with method calls on nullable receivers:

```slang
Node = class {
    val value: i64
    var next: *Node?

    getValue = (self: &Node) -> i64 {
        self.value
    }

    printValue = (self: &Node) {
        print(self.value)
    }
}

main = () {
    val node: *Node? = null

    // Safe navigation with non-void method: returns nullable type
    val result: i64? = node?.getValue()   // returns i64? (null if node is null)

    // Safe navigation with void method: no-op if null
    node?.printValue()                     // does nothing if node is null

    // Using Elvis operator for default values
    val value = node?.getValue() ?: 0      // returns 0 if node is null
}
```

**Rules:**
- `receiver?.method()` where method returns `T` yields `T?`
- `receiver?.method()` where method returns void is a no-op when receiver is null
- The Elvis operator `?:` can provide default values: `nullable ?: default`
- Elvis precedence is lower than arithmetic: `a ?: b + c` parses as `a ?: (b + c)`

## Instance vs Static Methods

Methods are distinguished by the presence of `self` as first parameter:

- **Instance method:** First parameter is `self` with a receiver type. Called on instances.
- **Static method:** No `self` parameter. Called on the class name.

```slang
Point = class {
    val x: i64
    val y: i64

    // Instance method - has self, called on instance
    magnitude = (self: &Point) -> i64 {
        self.x * self.x + self.y * self.y
    }

    // Static method - no self, called on class
    origin = () -> Point {
        Point{ 0, 0 }
    }
}

main = () {
    val p = Point{ 3, 4 }
    p.magnitude()      // instance method call
    Point.origin()     // static method call
}
```

## Method Overloading

Methods can be overloaded by having multiple methods with the same name but different parameter signatures:

```slang
Point = class {
    val x: i64
    val y: i64

    // Overloaded 'distance' methods
    distance = (self: &Point) -> i64 {
        // Distance from origin
        self.x * self.x + self.y * self.y
    }

    distance = (self: &Point, other: &Point) -> i64 {
        // Distance from another point
        val dx = self.x - other.x
        val dy = self.y - other.y
        dx * dx + dy * dy
    }
}

main = () {
    val p1 = Point{ 3, 4 }
    val p2 = Point{ 6, 8 }

    print(p1.distance())      // calls distance(self) -> 25
    print(p1.distance(p2))    // calls distance(self, other) -> 25
}
```

**Overloading rules:**
- Methods are distinguished by parameter count and types (not return type)
- The compiler selects the most specific matching overload (follows Kotlin)
- Ambiguous calls result in a compile-time error

**Specificity rules (Kotlin-style):**
- Non-nullable types are more specific than nullable types (`i64` > `i64?`)
- When multiple overloads are applicable, the most specific one is selected
- If no single most-specific overload exists, the call is ambiguous (compile error)

```slang
Printer = class {
    // Overloaded by nullable vs non-nullable
    print = (self: &Printer, x: i64) {
        // handles non-null
    }

    print = (self: &Printer, x: i64?) {
        // handles nullable
    }
}

main = () {
    val p = Printer{}
    val num: i64 = 42
    val maybe: i64? = 42

    p.print(num)    // calls i64 version (more specific)
    p.print(maybe)  // calls i64? version (only one applicable)
    p.print(null)   // calls i64? version (only one applicable)
}
```

## Method Receiver Types (SEP 1)

Using the implemented pointer syntax from SEP 1:

- `self: &T` - Immutable borrow; read-only access, caller keeps ownership.
- `self: &&T` - Mutable borrow; can modify `var` fields, caller keeps ownership.
- `self: *T` - Takes ownership; instance is consumed after the method call.

**Pointer Syntax Rationale:**
- `&T` - Immutable borrow (like Rust's `&T`)
- `&&T` - Mutable borrow (like Rust's `&mut T`). The double-ampersand was chosen to avoid introducing a `mut` keyword while keeping the syntax concise.
- `*T` - Owned pointer (like Rust's `Box<T>` or C's `T*`)

Note: The `var` modifier on parameters controls reassignability of the parameter binding itself, not borrow mutability. Use `&&T` for mutable borrow access.

## Stack vs Heap Allocation

Class instances can be allocated on the stack (local variable) or heap (via `Heap.new()`):

```slang
// Stack-allocated: lives until end of current scope
val stackPoint = Point{ 10, 20 }

// Heap-allocated: lives until ownership is released
val heapPoint = Heap.new(Point{ 10, 20 })  // heapPoint: *Point
```

**When to use each:**

| Allocation | Syntax | Type | Use When |
|------------|--------|------|----------|
| Stack | `Point{ ... }` | `Point` | Short-lived, local values; no ownership transfer needed |
| Heap | `Heap.new(Point{ ... })` | `*Point` | Values that outlive current scope, or when ownership transfer is needed |

**Method compatibility:**

- Stack values auto-borrow for `&T` and `&&T` methods
- Stack values **cannot** call `*T` (consuming) methods - use `Heap.new()` if consumption is needed
- Heap values (`*T`) work with all receiver types

```slang
Point = class {
    read = (self: &Point) { ... }      // OK on stack and heap
    mutate = (self: &&Point) { ... }   // OK on stack and heap
    consume = (self: *Point) { ... }   // Only works on *Point (heap)
}

main = () {
    val stack = Point{ 1, 2 }
    stack.read()     // OK: auto-borrows as &Point
    stack.mutate()   // OK: auto-borrows as &&Point
    // stack.consume() // Error: cannot consume stack-allocated value

    val heap = Heap.new(Point{ 1, 2 })
    heap.read()      // OK: auto-borrows as &Point
    heap.mutate()    // OK: auto-borrows as &&Point
    heap.consume()   // OK: transfers ownership, heap is moved
}
```

## Returning Class Instances by Value

When a function returns a class type (not a pointer), the value is **copied** to the caller:

```slang
Point = class {
    val x: i64
    val y: i64

    // Returns a new Point by value (copied to caller)
    origin = () -> Point {
        Point{ 0, 0 }
    }

    // Returns a copy of self's data
    clone = (self: &Point) -> Point {
        Point{ self.x, self.y }
    }
}

main = () {
    val p1 = Point.origin()  // p1 receives a copy
    val p2 = p1.clone()      // p2 receives a copy of p1
}
```

For owned pointers (`*T`), returning transfers ownership:

```slang
createPoint = () -> *Point {
    Heap.new(Point{ 10, 20 })  // ownership transferred to caller
}
```

# Description

## Relationship to Structs

Classes extend structs by adding methods. A class is essentially a struct with associated functions. The key differences:

| Feature | Struct | Class |
|---------|--------|-------|
| Fields | Yes | Yes |
| Methods | No | Yes |
| Static Methods | No | Yes |
| Instantiation | `StructName{ ... }` | `ClassName{ ... }` (same syntax) |
| Factory Methods | N/A | Via static methods (e.g., `ClassName.new(...)`) |

**When to use struct vs class:**

- Use `struct` for pure data containers with no associated behavior (DTOs, records, tuples)
- Use `class` when the type has meaningful operations beyond field access
- If unsure, start with `struct` - converting to `class` later only requires changing the keyword

```slang
// Struct: pure data, no behavior
Coordinate = struct {
    val x: i64
    val y: i64
}

// Class: data with meaningful operations
Point = class {
    val x: i64
    val y: i64

    distanceFromOrigin = (self: &Point) -> i64 {
        self.x * self.x + self.y * self.y
    }
}
```

Note: Structs and classes have identical memory layout. The only difference is that classes can have methods.

## Step 1: Lexer Changes

**File:** `compiler/lexer/lexer.go`

Add token support for class syntax:

```go
// New token types
TokenTypeClass   // 'class' keyword
TokenTypeSelf    // 'self' keyword
TokenTypeObject  // 'object' keyword
```

Add to keywords map:
```go
"class":  TokenTypeClass,
"self":   TokenTypeSelf,
"object": TokenTypeObject,
```

## Step 2: AST Changes

**File:** `compiler/ast/ast.go`

Add new AST nodes for class declarations:

```go
// ClassDecl represents a class declaration
type ClassDecl struct {
    Name    string
    NamePos Position
    Fields  []FieldDecl      // Same as struct fields
    Methods []MethodDecl     // All methods (static determined by presence of 'self' param)
}

// ObjectDecl represents a singleton object declaration (static methods only)
type ObjectDecl struct {
    Name    string
    NamePos Position
    Methods []MethodDecl     // All methods must be static (no 'self' parameter)
}

// MethodDecl represents a method declaration within a class
type MethodDecl struct {
    Name       string
    NamePos    Position
    Params     []ParamDecl      // If first param is 'self', it's an instance method
    ReturnType string           // Empty for void
    ReturnPos  Position
    Body       *BlockStatement
}

// Note: Static vs instance is determined by checking if Params[0].Name == "self"

// For methods, the first parameter is 'self' with an explicit type:
//   self: &T           - immutable borrow
//   self: &&T          - mutable borrow
//   self: *T           - takes ownership
// The receiver type is parsed as a regular ParamDecl with name "self"

// SelfExpr represents the 'self' keyword in method bodies
type SelfExpr struct {
    Pos Position
}

// MethodCallExpr represents a method call on an instance
type MethodCallExpr struct {
    Object    Expression       // The receiver instance
    Method    string
    MethodPos Position
    Args      []Expression
}
```

## Step 3: Parser Changes

**File:** `compiler/parser/parser.go`

### Parse Class Declaration

At top-level, when encountering `<Name> = class {`:

```go
func (p *Parser) parseClassDecl() *ast.ClassDecl {
    // Current token is identifier (class name)
    name := p.currentToken.Value
    namePos := p.currentToken.Position

    p.advance() // consume name
    p.expect(TokenTypeAssign) // '='
    p.expect(TokenTypeClass)  // 'class'
    p.expect(TokenTypeLBrace) // '{'

    var fields []ast.FieldDecl
    var methods []ast.MethodDecl

    for p.currentToken.Type != TokenTypeRBrace {
        if p.currentToken.Type == TokenTypeVal ||
           p.currentToken.Type == TokenTypeVar {
            field := p.parseFieldDecl()
            fields = append(fields, field)
        } else {
            // Parse method - static vs instance determined by presence of 'self' param
            method := p.parseMethodDecl()
            methods = append(methods, method)
        }
    }

    p.expect(TokenTypeRBrace) // '}'

    return &ast.ClassDecl{
        Name:    name,
        NamePos: namePos,
        Fields:  fields,
        Methods: methods,
    }
}
```

Note: Whether a method is static or instance is determined during semantic analysis by checking if the first parameter is named `self`.

### Parse Method Declaration

Methods use the same assignment syntax as functions:

```go
func (p *Parser) parseMethodDecl() ast.MethodDecl {
    // Pattern: methodName = (params) -> ReturnType { body }
    //      or: methodName = (params) { body }  (void return)

    name := p.currentToken.Value
    namePos := p.currentToken.Position

    p.advance()               // consume name
    p.expect(TokenTypeAssign) // '='
    p.expect(TokenTypeLParen) // '('

    params := p.parseParamList()

    p.expect(TokenTypeRParen) // ')'

    var returnType string
    var returnPos Position
    if p.currentToken.Type == TokenTypeArrow {
        p.advance()
        returnType = p.parseTypeName()
        returnPos = p.currentToken.Position
    }

    body := p.parseBlockStatement()

    return ast.MethodDecl{
        Name:       name,
        NamePos:    namePos,
        Params:     params,
        ReturnType: returnType,
        ReturnPos:  returnPos,
        Body:       body,
    }
}
```

### Parse Method Call Expression

In the expression parser, when parsing postfix operators after a dot:

```go
// After parsing: expr.identifier
// Check if followed by '(' to determine field access vs method call
if p.currentToken.Type == TokenTypeLParen {
    // Method call
    p.advance() // consume '('
    args := p.parseArgList()
    p.expect(TokenTypeRParen)

    expr = &ast.MethodCallExpr{
        Object:    expr,
        Method:    fieldName,
        MethodPos: fieldPos,
        Args:      args,
    }
} else {
    // Field access (existing behavior)
    expr = &ast.FieldAccessExpr{...}
}
```

### Parse Static Method Call

Static method calls like `Point.origin()` are parsed as regular `MethodCallExpr` where the `Object` is an `IdentifierExpr` containing the class name. The distinction between instance and static calls is made during **semantic analysis**, not parsing.

```go
// Point.origin() parses as:
// MethodCallExpr {
//     Object: IdentifierExpr{Name: "Point"},  // class name as identifier
//     Method: "origin",
//     Args:   [],
// }

// p.magnitude() parses identically:
// MethodCallExpr {
//     Object: IdentifierExpr{Name: "p"},      // variable name as identifier
//     Method: "magnitude",
//     Args:   [],
// }
```

During semantic analysis, when analyzing a `MethodCallExpr`:
1. Analyze the `Object` expression to get its type
2. If the object is an identifier that resolves to a **class type** (not a variable), it's a static method call
3. If the object resolves to a **variable** of class type (or pointer to class), it's an instance method call
4. Validate that the method exists and matches the call type (static vs instance)

```go
func (a *Analyzer) analyzeMethodCallExpr(expr *ast.MethodCallExpr) TypedExpression {
    // Check if Object is an identifier that names a class (static call)
    if ident, ok := expr.Object.(*ast.IdentifierExpr); ok {
        if classType, isClass := a.types[ident.Name].(ClassType); isClass {
            // Static method call: ClassName.method()
            return a.analyzeStaticMethodCall(classType, expr)
        }
    }

    // Otherwise, analyze as instance method call
    typedObject := a.analyzeExpression(expr.Object)
    // ... rest of instance method call analysis
}
```

### Parse Self Expression

In primary expression parsing:

```go
case TokenTypeSelf:
    pos := p.currentToken.Position
    p.advance()
    return &ast.SelfExpr{Pos: pos}
```

## Step 4: Type System Changes

**File:** `compiler/semantic/types.go`

Add class type representation:

```go
// ClassType represents a class with fields and methods
type ClassType struct {
    Name    string
    Fields  []FieldInfo
    Methods map[string][]*MethodInfo  // All methods, grouped by name (supports overloading)
}

// MethodInfo contains method signature information
type MethodInfo struct {
    Name       string
    ParamTypes []Type        // Includes self type as first element for instance methods
    ReturnType Type
    IsStatic   bool          // Derived: true if first param is not 'self'
}

// For non-static methods, ParamTypes[0] is the self type:
//   RefPointerType{Inner: ClassType}       - self: &T (immutable borrow)
//   MutRefPointerType{Inner: ClassType}    - self: &&T (mutable borrow)
//   OwnedPointerType{Inner: ClassType}     - self: *T (takes ownership)

func (t ClassType) String() string {
    return t.Name
}

func (t ClassType) Equals(other Type) bool {
    o, ok := other.(ClassType)
    if !ok {
        return false
    }
    return t.Name == o.Name
}

// ObjectType represents a singleton object with static methods only
type ObjectType struct {
    Name    string
    Methods map[string][]*MethodInfo  // All methods must be static (no self parameter)
}

func (t ObjectType) String() string {
    return t.Name
}

func (t ObjectType) Equals(other Type) bool {
    o, ok := other.(ObjectType)
    if !ok {
        return false
    }
    return t.Name == o.Name
}
```

## Step 5: Semantic Analysis

**File:** `compiler/semantic/analyzer.go`

### Register Class Types

During declaration phase, register class types:

```go
func (a *Analyzer) registerClassType(decl *ast.ClassDecl) {
    // Check for duplicate type name
    if _, exists := a.types[decl.Name]; exists {
        a.addError(...)
        return
    }

    // Parse fields
    fields := make([]FieldInfo, len(decl.Fields))
    for i, f := range decl.Fields {
        fields[i] = FieldInfo{
            Name:    f.Name,
            Type:    a.resolveTypeName(f.TypeName, f.TypePos),
            Mutable: f.Mutable,
        }
    }

    // Parse methods - determine static by checking first param
    // Use slice per name to support overloading
    methods := make(map[string][]*MethodInfo)
    for _, m := range decl.Methods {
        paramTypes := a.resolveParamTypes(m.Params)
        isStatic := len(m.Params) == 0 || m.Params[0].Name != "self"

        methodInfo := &MethodInfo{
            Name:       m.Name,
            ParamTypes: paramTypes,
            ReturnType: a.resolveTypeName(m.ReturnType, m.ReturnPos),
            IsStatic:   isStatic,
        }
        methods[m.Name] = append(methods[m.Name], methodInfo)
    }

    a.types[decl.Name] = ClassType{
        Name:    decl.Name,
        Fields:  fields,
        Methods: methods,
    }
}
```

### Register Object Types

During declaration phase, register object types with validation:

```go
func (a *Analyzer) registerObjectType(decl *ast.ObjectDecl) {
    // Check for duplicate type name (shares namespace with classes, structs, functions)
    if _, exists := a.types[decl.Name]; exists {
        a.addError("type '%s' is already defined", decl.Name)
        return
    }

    // Objects cannot have fields - this is enforced at parse time
    // (ObjectDecl doesn't have a Fields member)

    // Parse methods - all must be static (no self parameter)
    methods := make(map[string][]*MethodInfo)
    for _, m := range decl.Methods {
        // Validate no self parameter
        if len(m.Params) > 0 && m.Params[0].Name == "self" {
            a.addError("object methods cannot have 'self' parameter; method '%s' in object '%s'",
                m.Name, decl.Name)
            continue
        }

        paramTypes := a.resolveParamTypes(m.Params)

        methodInfo := &MethodInfo{
            Name:       m.Name,
            ParamTypes: paramTypes,
            ReturnType: a.resolveTypeName(m.ReturnType, m.ReturnPos),
            IsStatic:   true,  // Always static for objects
        }
        methods[m.Name] = append(methods[m.Name], methodInfo)
    }

    a.types[decl.Name] = ObjectType{
        Name:    decl.Name,
        Methods: methods,
    }
}
```

### Analyze Method Bodies

When analyzing method bodies, inject `self` into scope:

```go
func (a *Analyzer) analyzeMethodDecl(classType ClassType, method *ast.MethodDecl) *TypedMethodDecl {
    // Create method scope
    a.pushScope()
    defer a.popScope()

    // Validate self position: if any param is named 'self', it must be first
    for i, param := range method.Params {
        if param.Name == "self" && i != 0 {
            a.addError("'self' must be the first parameter")
        }
    }

    // Add parameters to scope (including 'self' for non-static methods)
    for _, param := range method.Params {
        paramType := a.resolveTypeName(param.TypeName, param.TypePos)

        // For 'self' parameter, validate and determine mutability
        isSelf := param.Name == "self"
        mutable := param.Mutable  // 'var' modifier on parameter

        if isSelf {
            // Validate that self type references this class
            if err := a.validateSelfType(paramType, classType, param.TypePos); err != nil {
                a.addError(err)
            }

            // Track ownership for 'self' (SEP 1)
            // self: *T means method takes ownership
            if _, isOwned := paramType.(OwnedPointerType); isOwned {
                a.currentMethodTakesOwnership = true
            }
        }

        a.currentScope.Define(param.Name, VariableInfo{
            Type:    paramType,
            Mutable: mutable,
        })
    }

    // Analyze body
    typedBody := a.analyzeBlockStatement(method.Body)

    // Check return type
    // ...

    return &TypedMethodDecl{...}
}

// validateSelfType ensures the self parameter type references the enclosing class
func (a *Analyzer) validateSelfType(selfType Type, classType ClassType, pos ast.Position) error {
    // Extract inner type from pointer wrapper
    var innerType Type
    switch t := selfType.(type) {
    case RefPointerType:
        innerType = t.Inner
    case MutRefPointerType:
        innerType = t.Inner
    case OwnedPointerType:
        innerType = t.Inner
    default:
        return fmt.Errorf("'self' must have pointer type (&%s, &&%s, or *%s), got %s",
            classType.Name, classType.Name, classType.Name, selfType)
    }

    // Check that inner type matches the enclosing class
    if inner, ok := innerType.(ClassType); ok {
        if inner.Name != classType.Name {
            return fmt.Errorf("'self' type must reference enclosing class '%s', got '%s'",
                classType.Name, inner.Name)
        }
        return nil
    }

    return fmt.Errorf("'self' type must reference class '%s', got %s",
        classType.Name, innerType)
}
```

### Analyze Method Calls

Method calls are dispatched to either instance or static analysis based on whether the object is a class name or a value:

```go
func (a *Analyzer) analyzeMethodCallExpr(expr *ast.MethodCallExpr) TypedExpression {
    // Check if Object is an identifier that names a class (static call)
    if ident, ok := expr.Object.(*ast.IdentifierExpr); ok {
        if classType, isClass := a.types[ident.Name].(ClassType); isClass {
            return a.analyzeStaticMethodCall(classType, expr)
        }
    }

    // Instance method call - analyze the receiver
    typedObject := a.analyzeExpression(expr.Object)
    objectType := typedObject.ExprType()

    // Check object is a class type (or pointer to class)
    classType, ok := objectType.(ClassType)
    if !ok {
        // Also check for *Class or &Class (SEP 1)
        if ptrType, ok := objectType.(OwnedPointerType); ok {
            classType, ok = ptrType.Inner.(ClassType)
        } else if refType, ok := objectType.(RefPointerType); ok {
            classType, ok = refType.Inner.(ClassType)
        } else if mutRefType, ok := objectType.(MutRefPointerType); ok {
            classType, ok = mutRefType.Inner.(ClassType)
        }
        if !ok {
            a.addError("cannot call method on non-class type %s", objectType)
            return errorExpr(expr)
        }
    }

    // Look up method overloads
    overloads, exists := classType.Methods[expr.Method]
    if !exists || len(overloads) == 0 {
        a.addError("class %s has no method %s", classType.Name, expr.Method)
        return errorExpr(expr)
    }

    // Filter to instance methods only (have self parameter)
    instanceOverloads := filterInstanceMethods(overloads)
    if len(instanceOverloads) == 0 {
        a.addError("cannot call static method '%s' on instance; use '%s.%s()'",
            expr.Method, classType.Name, expr.Method)
        return errorExpr(expr)
    }

    // Resolve overload based on argument types (Kotlin-style specificity)
    methodInfo := a.resolveOverload(instanceOverloads, expr.Args)
    if methodInfo == nil {
        a.addError("no matching overload for method '%s' with arguments (%s)\n  available overloads:\n%s",
            expr.Method, formatArgTypes(expr.Args), formatOverloads(instanceOverloads))
        return errorExpr(expr)
    }

    // Check self parameter type for ownership (SEP 1)
    if len(methodInfo.ParamTypes) > 0 {
        selfType := methodInfo.ParamTypes[0]
        if _, isOwned := selfType.(OwnedPointerType); isOwned {
            // Method takes ownership - mark receiver as moved
            a.markAsMoved(expr.Object)
        }
    }

    // Type check arguments (skip self, it's the receiver)
    typedArgs := a.analyzeArguments(expr.Args, methodInfo.ParamTypes[1:])

    return &TypedMethodCallExpr{
        Type:           methodInfo.ReturnType,
        Object:         typedObject,
        Method:         expr.Method,
        Args:           typedArgs,
        ResolvedMethod: methodInfo,
    }
}

// analyzeStaticMethodCall handles ClassName.method() calls
func (a *Analyzer) analyzeStaticMethodCall(classType ClassType, expr *ast.MethodCallExpr) TypedExpression {
    // Look up method overloads
    overloads, exists := classType.Methods[expr.Method]
    if !exists || len(overloads) == 0 {
        a.addError("class %s has no method %s", classType.Name, expr.Method)
        return errorExpr(expr)
    }

    // Filter to static methods only (no self parameter)
    staticOverloads := filterStaticMethods(overloads)
    if len(staticOverloads) == 0 {
        a.addError("cannot call instance method '%s' without a receiver; use 'instance.%s()'",
            expr.Method, expr.Method)
        return errorExpr(expr)
    }

    // Resolve overload based on argument types (Kotlin-style specificity)
    methodInfo := a.resolveOverload(staticOverloads, expr.Args)
    if methodInfo == nil {
        a.addError("no matching overload for method '%s' with arguments (%s)\n  available overloads:\n%s",
            expr.Method, formatArgTypes(expr.Args), formatOverloads(staticOverloads))
        return errorExpr(expr)
    }

    // Type check all arguments (no self to skip)
    typedArgs := a.analyzeArguments(expr.Args, methodInfo.ParamTypes)

    // Create a TypedMethodCallExpr with the class name as the Object
    // (static calls use the same struct as instance calls)
    return &TypedMethodCallExpr{
        Type:           methodInfo.ReturnType,
        Object:         &TypedIdentifierExpr{Type: classType, Name: classType.Name},
        Method:         expr.Method,
        Arguments:      typedArgs,
        ResolvedMethod: methodInfo,
    }
}
```

### Overload Resolution

Overload resolution follows Kotlin-style specificity rules:

```go
// resolveOverload finds the most specific matching overload for the given arguments
// Returns nil if no match or ambiguous
func (a *Analyzer) resolveOverload(overloads []*MethodInfo, args []ast.Expression) *MethodInfo {
    // First, find all applicable overloads (correct arg count and compatible types)
    applicable := []*MethodInfo{}

    for _, method := range overloads {
        // For instance methods, skip self parameter in count
        expectedArgs := len(method.ParamTypes)
        if !method.IsStatic && expectedArgs > 0 {
            expectedArgs--  // don't count self
        }

        if len(args) != expectedArgs {
            continue
        }

        if a.argsCompatible(args, method) {
            applicable = append(applicable, method)
        }
    }

    if len(applicable) == 0 {
        return nil
    }

    if len(applicable) == 1 {
        return applicable[0]
    }

    // Multiple applicable - find most specific
    return a.mostSpecificOverload(applicable, args)
}

// argsCompatible checks if arguments are type-compatible with method parameters
func (a *Analyzer) argsCompatible(args []ast.Expression, method *MethodInfo) bool {
    paramTypes := method.ParamTypes
    startIdx := 0
    if !method.IsStatic && len(paramTypes) > 0 {
        startIdx = 1  // skip self
    }

    for i, arg := range args {
        paramType := paramTypes[startIdx+i]
        argType := a.inferExprType(arg)

        if !a.isAssignable(argType, paramType) {
            return false
        }
    }
    return true
}

// mostSpecificOverload applies Kotlin-style specificity rules
// Non-nullable is more specific than nullable
func (a *Analyzer) mostSpecificOverload(candidates []*MethodInfo, args []ast.Expression) *MethodInfo {
    // Compare each pair of candidates
    var best *MethodInfo = nil

    for _, candidate := range candidates {
        if best == nil {
            best = candidate
            continue
        }

        cmp := a.compareSpecificity(candidate, best)
        if cmp > 0 {
            best = candidate
        } else if cmp == 0 {
            // Ambiguous - neither is more specific
            return nil
        }
    }

    // Verify best is more specific than ALL others
    for _, candidate := range candidates {
        if candidate == best {
            continue
        }
        if a.compareSpecificity(best, candidate) <= 0 {
            return nil  // ambiguous
        }
    }

    return best
}

// compareSpecificity returns:
//   +1 if a is more specific than b
//   -1 if b is more specific than a
//    0 if neither is more specific (ambiguous)
func (a *Analyzer) compareSpecificity(methodA, methodB *MethodInfo) int {
    paramsA := methodA.ParamTypes
    paramsB := methodB.ParamTypes

    startA, startB := 0, 0
    if !methodA.IsStatic { startA = 1 }
    if !methodB.IsStatic { startB = 1 }

    aMoreSpecific := false
    bMoreSpecific := false

    for i := 0; i < len(paramsA)-startA; i++ {
        typeA := paramsA[startA+i]
        typeB := paramsB[startB+i]

        // Non-nullable is more specific than nullable
        aNullable := isNullableType(typeA)
        bNullable := isNullableType(typeB)

        if !aNullable && bNullable {
            aMoreSpecific = true
        } else if aNullable && !bNullable {
            bMoreSpecific = true
        }
    }

    if aMoreSpecific && !bMoreSpecific {
        return 1
    } else if bMoreSpecific && !aMoreSpecific {
        return -1
    }
    return 0
}

// Helper functions for filtering and formatting

func filterInstanceMethods(methods []*MethodInfo) []*MethodInfo {
    result := []*MethodInfo{}
    for _, m := range methods {
        if !m.IsStatic {
            result = append(result, m)
        }
    }
    return result
}

func filterStaticMethods(methods []*MethodInfo) []*MethodInfo {
    result := []*MethodInfo{}
    for _, m := range methods {
        if m.IsStatic {
            result = append(result, m)
        }
    }
    return result
}

func formatArgTypes(args []ast.Expression) string {
    // Returns comma-separated list of inferred arg types
    // e.g., "i64, string, bool"
    types := []string{}
    for _, arg := range args {
        types = append(types, inferType(arg).String())
    }
    return strings.Join(types, ", ")
}

func formatOverloads(methods []*MethodInfo) string {
    // Returns formatted list of overload signatures
    lines := []string{}
    for _, m := range methods {
        sig := formatMethodSignature(m)
        lines = append(lines, "    "+sig)
    }
    return strings.Join(lines, "\n")
}

func formatMethodSignature(m *MethodInfo) string {
    params := []string{}
    for i, p := range m.ParamTypes {
        if i == 0 && !m.IsStatic {
            params = append(params, fmt.Sprintf("self: %s", p))
        } else {
            params = append(params, p.String())
        }
    }
    if m.ReturnType != nil {
        return fmt.Sprintf("%s(%s) -> %s", m.Name, strings.Join(params, ", "), m.ReturnType)
    }
    return fmt.Sprintf("%s(%s)", m.Name, strings.Join(params, ", "))
}

func isNullableType(t Type) bool {
    _, ok := t.(NullableType)
    return ok
}
```

### Analyze Self Expression

```go
func (a *Analyzer) analyzeSelfExpr(expr *ast.SelfExpr) TypedExpression {
    // Look up 'self' in current scope
    selfInfo, exists := a.currentScope.Lookup("self")
    if !exists {
        a.addError("'self' can only be used inside a method")
        return errorExpr(expr)
    }

    return &TypedSelfExpr{
        Type: selfInfo.Type,
        Pos:  expr.Pos,
    }
}
```

## Step 6: Typed AST

**File:** `compiler/semantic/typed_ast.go`

Add typed nodes:

```go
type TypedClassDecl struct {
    Name      string
    ClassType ClassType
    Fields    []TypedFieldDecl
    Methods   []*TypedMethodDecl
}

type TypedMethodDecl struct {
    Name       string
    Params     []TypedParamDecl
    ReturnType Type
    Body       *TypedBlockStatement
    IsStatic   bool    // Derived from whether first param is 'self'
}

// TypedMethodCallExpr represents both instance and static method calls.
// For instance calls: Object is the receiver expression (e.g., p in p.method())
// For static calls: Object is an identifier expression with the class/object name
type TypedMethodCallExpr struct {
    Type           Type              // the return type of the method
    Object         TypedExpression   // the receiver expression (instance or class name)
    Dot            ast.Position      // position of '.' or '?.'
    Method         string            // method name
    MethodPos      ast.Position      // position of method name
    LeftParen      ast.Position      // position of '('
    Arguments      []TypedExpression // typed argument expressions
    RightParen     ast.Position      // position of ')'
    ResolvedMethod *MethodInfo       // the resolved method (for overload resolution)
    SafeNavigation bool              // true for ?.method() (safe navigation on nullable)
}

type TypedSelfExpr struct {
    Type    Type
    SelfPos ast.Position
}

func (e *TypedMethodCallExpr) ExprType() Type { return e.Type }
func (e *TypedSelfExpr) ExprType() Type       { return e.Type }
```

Note: The implementation uses a unified `TypedMethodCallExpr` for both static and instance method calls. Static calls are distinguished by `ResolvedMethod.IsStatic` being true, and the `Object` field contains an identifier expression with the class/object name.

## Step 7: Code Generation

**File:** `compiler/codegen/typed_codegen.go`

### Instance Layout

Class instances have the same memory layout as structs (just fields, no vtable needed since all dispatch is static):

```
+0:  field1 (8 bytes)
+8:  field2 (8 bytes)
+16: field3 (8 bytes)
...
```

This is identical to struct layout. Methods are called by mangled name, not through a vtable.

### Method Name Mangling

To support overloading, method names are mangled to include parameter types:

```go
// mangleMethodName creates a unique label for a method including parameter types
func (g *TypedCodeGenerator) mangleMethodName(className string, method *MethodInfo) string {
    // Start with _ClassName_methodName
    label := fmt.Sprintf("_%s_%s", className, method.Name)

    // Append parameter types (skip self for instance methods)
    startIdx := 0
    if !method.IsStatic && len(method.ParamTypes) > 0 {
        startIdx = 1  // skip self parameter
    }

    for i := startIdx; i < len(method.ParamTypes); i++ {
        paramType := method.ParamTypes[i]
        label += "_" + mangleType(paramType)
    }

    return label
}

// mangleType converts a type to a string suitable for name mangling
func mangleType(t Type) string {
    switch tt := t.(type) {
    case IntegerType:
        return tt.Name  // "i64", "i32", etc.
    case BooleanType:
        return "bool"
    case StringType:
        return "string"
    case ClassType:
        return tt.Name
    case RefPointerType:
        return "Ref" + mangleType(tt.Inner)
    case MutRefPointerType:
        return "MutRef" + mangleType(tt.Inner)
    case OwnedPointerType:
        return "Ptr" + mangleType(tt.Inner)
    case NullableType:
        return "Nullable_" + mangleType(tt.Inner)
    default:
        return "unknown"
    }
}
```

**Examples:**
| Method Signature | Mangled Name |
|------------------|--------------|
| `distance(self: &Point)` | `_Point_distance` |
| `distance(self: &Point, other: &Point)` | `_Point_distance_RefPoint` |
| `process(self: &Printer, x: i64)` | `_Printer_process_i64` |
| `process(self: &Printer, x: i64?)` | `_Printer_process_Nullable_i64` |
| `create(x: i64, y: i64)` (static) | `_Point_create_i64_i64` |

### Generate Method Code

Methods are generated as regular functions with mangled names. Instance methods receive `self` in x0, while static methods receive their first argument in x0:

```go
func (g *TypedCodeGenerator) generateMethodDecl(className string, method *TypedMethodDecl) string {
    builder := strings.Builder{}

    // Method label: _ClassName_methodName_ParamTypes (for overload uniqueness)
    // Examples:
    //   distance(self: &Point) -> _Point_distance
    //   distance(self: &Point, other: &Point) -> _Point_distance_RefPoint
    //   process(self: &Printer, x: i64) -> _Point_process_i64
    //   process(self: &Printer, x: i64?) -> _Point_process_Nullable_i64
    methodLabel := g.mangleMethodName(className, method)
    builder.WriteString(fmt.Sprintf(".global %s\n", methodLabel))
    builder.WriteString(fmt.Sprintf("%s:\n", methodLabel))

    // Prologue
    builder.WriteString("    stp x29, x30, [sp, #-16]!\n")
    builder.WriteString("    mov x29, sp\n")

    if method.IsStatic {
        // Static method: params in x0, x1, x2, ...
        for i := range method.Params {
            reg := fmt.Sprintf("x%d", i)
            offset := (i + 1) * 16
            builder.WriteString(fmt.Sprintf("    str %s, [sp, #-%d]!\n", reg, offset))
        }
    } else {
        // Instance method: self in x0, other params in x1, x2, ...
        builder.WriteString("    str x0, [sp, #-16]!\n")  // self at [sp]

        // Store other parameters (skip self which is Params[0])
        for i := 1; i < len(method.Params); i++ {
            reg := fmt.Sprintf("x%d", i)  // x1, x2, x3...
            offset := (i + 1) * 16
            builder.WriteString(fmt.Sprintf("    str %s, [sp, #-%d]!\n", reg, offset))
        }
    }

    // Generate body
    bodyCode, _ := g.generateBlockStatement(method.Body, ctx)
    builder.WriteString(bodyCode)

    // Epilogue
    builder.WriteString("    mov sp, x29\n")
    builder.WriteString("    ldp x29, x30, [sp], #16\n")
    builder.WriteString("    ret\n")

    return builder.String()
}
```

### Generate Instance Method Call

```go
func (g *TypedCodeGenerator) generateInstanceMethodCall(expr *TypedMethodCallExpr, ctx *CodeGenContext) (string, error) {
    builder := strings.Builder{}

    // Evaluate receiver into x0
    objCode, err := g.generateExpr(expr.Object, ctx)
    if err != nil {
        return "", err
    }
    builder.WriteString(objCode)
    builder.WriteString("    mov x0, x2\n")  // receiver in x0

    // Evaluate arguments into x1, x2, x3, ...
    for i, arg := range expr.Args {
        argCode, err := g.generateExpr(arg, ctx)
        if err != nil {
            return "", err
        }
        builder.WriteString(argCode)
        reg := fmt.Sprintf("x%d", i+1)
        builder.WriteString(fmt.Sprintf("    mov %s, x2\n", reg))
    }

    // Get class type to find method (unwrap pointer types if needed)
    classType := getClassType(expr.Object.ExprType())
    // Use mangled name that includes parameter types for overload resolution
    methodLabel := g.mangleMethodName(classType.Name, expr.ResolvedMethod)

    // Call method
    builder.WriteString(fmt.Sprintf("    bl %s\n", methodLabel))

    // Result is in x0, move to x2 for consistency
    builder.WriteString("    mov x2, x0\n")

    return builder.String(), nil
}

// getClassType unwraps pointer types to get the underlying ClassType
func getClassType(t Type) ClassType {
    switch tt := t.(type) {
    case ClassType:
        return tt
    case OwnedPointerType:
        return getClassType(tt.Inner)
    case RefPointerType:
        return getClassType(tt.Inner)
    case MutRefPointerType:
        return getClassType(tt.Inner)
    default:
        panic("expected class type")
    }
}
```

### Dispatching Method Calls

In the code generator, `TypedMethodCallExpr` handles both static and instance calls. The dispatch is based on `ResolvedMethod.IsStatic`:

```go
func (g *TypedCodeGenerator) generateMethodCall(expr *TypedMethodCallExpr, ctx *CodeGenContext) (string, error) {
    if expr.ResolvedMethod.IsStatic {
        return g.generateStaticMethodCall(expr, ctx)
    }
    return g.generateInstanceMethodCall(expr, ctx)
}

// generateStaticMethodCall handles static method calls (ClassName.method())
// Arguments start at x0 (no receiver)
func (g *TypedCodeGenerator) generateStaticMethodCall(expr *TypedMethodCallExpr, ctx *CodeGenContext) (string, error) {
    builder := strings.Builder{}

    // Evaluate arguments into x0, x1, x2, ... (no receiver)
    for i, arg := range expr.Arguments {
        argCode, err := g.generateExpr(arg, ctx)
        if err != nil {
            return "", err
        }
        builder.WriteString(argCode)
        reg := fmt.Sprintf("x%d", i)  // starts at x0, not x1
        builder.WriteString(fmt.Sprintf("    mov %s, x2\n", reg))
    }

    // Get class name from the Object (which is a TypedIdentifierExpr for static calls)
    className := expr.Object.GetType().(ClassType).Name
    methodLabel := g.mangleMethodName(className, expr.ResolvedMethod)

    // Call method
    builder.WriteString(fmt.Sprintf("    bl %s\n", methodLabel))

    // Result is in x0, move to x2 for consistency
    builder.WriteString("    mov x2, x0\n")

    return builder.String(), nil
}
```

### Generate Self Access

```go
func (g *TypedCodeGenerator) generateSelfExpr(expr *TypedSelfExpr, ctx *CodeGenContext) (string, error) {
    // Self is stored at a known offset on the stack
    offset := ctx.SelfOffset
    return fmt.Sprintf("    ldr x2, [x29, #%d]\n", offset), nil
}
```

## Instance Construction

Class instances are constructed using the same struct-literal syntax as structs. No special `init` method is needed - construction logic belongs in static factory methods.

```go
func (g *TypedCodeGenerator) generateClassConstruction(className string, args []TypedExpression, ctx *CodeGenContext) (string, error) {
    builder := strings.Builder{}
    classType := g.types[className].(ClassType)

    // Calculate instance size (same as struct - just fields)
    instanceSize := len(classType.Fields) * 8

    // Allocate memory (stack or heap depending on context)
    builder.WriteString(fmt.Sprintf("    mov x0, #%d\n", instanceSize))
    builder.WriteString("    bl _sl_alloc\n")  // allocation helper
    builder.WriteString("    mov x9, x0\n")    // save instance ptr

    // Initialize fields from arguments (direct assignment, no init method)
    for i, arg := range args {
        argCode, _ := g.generateExpr(arg, ctx)
        builder.WriteString(argCode)
        offset := i * 8
        builder.WriteString(fmt.Sprintf("    str x2, [x9, #%d]\n", offset))
    }

    // Return instance pointer
    builder.WriteString("    mov x2, x9\n")

    return builder.String(), nil
}
```

## Static Factory Methods

Slang uses **static factory methods** for construction logic beyond simple field assignment. This pattern is explicit, requires no special `init` method, and is a proven approach used in Kotlin, Rust, and recommended in Effective Java.

```slang
Point = class {
    val x: i64
    val y: i64

    // Static factory methods provide named "constructors"
    // (no self parameter = static method)
    origin = () -> Point {
        Point{ 0, 0 }
    }

    fromX = (x: i64) -> Point {
        Point{ x, 0 }
    }

    fromY = (y: i64) -> Point {
        Point{ 0, y }
    }

    diagonal = (n: i64) -> Point {
        Point{ n, n }
    }
}

main = () {
    val p1 = Point{ 10, 20 }         // direct field initialization
    val p2 = Point.origin()          // static factory: (0, 0)
    val p3 = Point.fromX(5)          // static factory: (5, 0)
    val p4 = Point.fromY(7)          // static factory: (0, 7)
    val p5 = Point.diagonal(3)       // static factory: (3, 3)
}
```

### Derived Fields via Factory Methods

For classes with derived fields that need computation during construction, use a factory method:

```slang
Circle = class {
    val radius: i64
    val area: i64        // derived field

    // Factory method computes derived fields
    new = (r: i64) -> Circle {
        Circle{ r, r * r * 3 }    // calculate area inline
    }

    unit = () -> Circle {
        Circle.new(1)
    }

    withDiameter = (d: i64) -> Circle {
        Circle.new(d / 2)
    }
}

main = () {
    // Use factory for automatic area calculation
    val c1 = Circle.new(5)           // radius=5, area=75

    // Direct construction requires all fields
    val c2 = Circle{ 5, 75 }         // same result, but manual
}
```

### Why No `init` Method?

An `init` method would add complexity without significant benefit:
1. Static factory methods can do everything `init` does
2. Factory methods have explicit, self-documenting names
3. Direct construction (`ClassName{ ... }`) stays simple and predictable
4. No hidden post-construction behavior

This approach mirrors Rust, where `Type { field: value }` is raw construction and `Type::new(...)` handles any logic.

## Ownership and Classes (SEP 1 Integration)

SEP 1 (Pointers and Memory) is implemented. Classes integrate with the ownership model through explicit receiver types using the implemented pointer syntax.

### Method Receiver Types

Methods declare their relationship to `self` using the implemented pointer types:

```slang
Point = class {
    var x: i64
    var y: i64

    // Immutable borrow - cannot modify self
    // Returns squared magnitude (no sqrt built-in yet)
    magnitudeSquared = (self: &Point) -> i64 {
        self.x * self.x + self.y * self.y
    }

    // Mutable borrow - can modify self, caller keeps ownership
    scale = (self: &&Point, factor: i64) {
        self.x = self.x * factor
        self.y = self.y * factor
    }

    // Takes ownership - self is consumed after call
    consume = (self: *Point) -> i64 {
        self.x + self.y
    }   // self freed here
}

main = () {
    var p = Heap.new(Point{ 3, 4 })

    print(p.magnitudeSquared())       // borrows p (immutable), prints: 25
    p.scale(2)                        // borrows p (mutable)
    print(p.x)                        // prints: 6

    val sum = p.consume()             // p moved, consumed
    print(sum)                        // prints: 18 (6 + 12)
    // print(p.x)                     // Error: p was moved
}
```

### Receiver Type Summary

| Receiver | Syntax | Effect | Caller Ownership |
|----------|--------|--------|------------------|
| Immutable borrow | `self: &T` | Read-only access | Keeps ownership |
| Mutable borrow | `self: &&T` | Can modify `var` fields | Keeps ownership |
| Takes ownership | `self: *T` | Consumes instance | Loses access |

### Heap-Allocated Class Instances

Class instances can be allocated on the heap using `Heap.new()`:

```slang
Point = class {
    var x: i64
    var y: i64

    // Static factory returning heap-allocated instance
    new = (x: i64, y: i64) -> *Point {
        Heap.new(Point{ x, y })
    }

    // Static factory for origin
    origin = () -> *Point {
        Point.new(0, 0)
    }

    translate = (self: &&Point, dx: i64, dy: i64) {
        self.x = self.x + dx
        self.y = self.y + dy
    }
}

main = () {
    // Stack-allocated (direct construction)
    val stackPoint = Point{ 1, 2 }

    // Heap-allocated (via factory)
    val heapPoint = Point.new(10, 20)     // heapPoint: *Point
    heapPoint.translate(5, 5)
    print(heapPoint.x)                     // prints: 15
}                                          // heapPoint freed here
```

### Classes Containing Pointers

Class fields can be owned pointers:

```slang
Node = class {
    val value: i64
    var next: *Node?

    new = (value: i64) -> *Node {
        Heap.new(Node{ value, null })
    }

    // Takes ownership of next node
    setNext = (self: &&Node, next: *Node) {
        self.next = next                   // ownership transferred to field
    }

    // Borrows to traverse
    printAll = (self: &Node) {
        print(self.value)
        self.next?.printAll()              // no-op if next is null (void method)
    }
}

main = () {
    var n1 = Node.new(10)
    var n2 = Node.new(20)
    var n3 = Node.new(30)

    n2.setNext(n3)                         // n3 moved into n2.next
    n1.setNext(n2)                         // n2 moved into n1.next

    n1.printAll()                          // prints: 10, 20, 30
}                                          // n1 freed, recursively frees chain
```

### Passing Class Instances to Functions

Functions accepting class instances use ownership types:

```slang
Point = class {
    var x: i64
    var y: i64
}

// Immutable borrow - read only
printPoint = (p: &Point) {
    print(p.x)
    print(p.y)
}

// Mutable borrow - can modify
scalePoint = (p: &&Point, factor: i64) {
    p.x = p.x * factor
    p.y = p.y * factor
}

// Takes ownership - consumes the instance
consume = (p: *Point) {
    print(p.x)
}                                          // p freed here

main = () {
    var p = Heap.new(Point{ 10, 20 })

    printPoint(p)                          // borrows
    scalePoint(p, 2)                       // mutably borrows
    print(p.x)                             // prints: 20

    consume(p)                             // ownership transferred
    // print(p.x)                          // Error: p was moved
}
```

## Error Handling

```slang
// Using self outside a method
main = () {
    print(self.x)                   // Error: 'self' can only be used inside a method
}

// Calling non-existent method
Counter = class {
    var count: i64
}
main = () {
    val c = Counter{ 0 }
    c.reset()                       // Error: class Counter has no method 'reset'
}

// Wrong argument count
Counter = class {
    var count: i64
    add = (self: &&Counter, n: i64) {
        self.count = self.count + n
    }
}
main = () {
    val c = Counter{ 0 }
    c.add()                         // Error: method 'add' expects 1 argument, got 0
    c.add(1, 2)                     // Error: method 'add' expects 1 argument, got 2
}

// Type mismatch in method call
main = () {
    val c = Counter{ 0 }
    c.add("hello")                  // Error: argument 1 of method 'add' expects i64, got string
}

// Calling instance method as static
Counter = class {
    var count: i64
    increment = (self: &&Counter) {
        self.count = self.count + 1
    }
}
main = () {
    Counter.increment()             // Error: cannot call instance method 'increment' without receiver
}

// Self not in first position
Counter = class {
    var count: i64
    add = (n: i64, self: &&Counter) {   // Error: 'self' must be the first parameter
        self.count = self.count + n
    }
}

// Trying to instantiate an object
Math = object {
    max = (a: i64, b: i64) -> i64 { ... }
}
main = () {
    val m = Math{}                      // Error: cannot instantiate object 'Math'
}

// Trying to add fields to an object
Utils = object {
    val value: i64                      // Error: objects cannot have fields
}

// Trying to add instance method to an object
Utils = object {
    getValue = (self: &Utils) -> i64 {  // Error: object methods cannot have 'self' parameter
        0
    }
}

// Ambiguous method overload
Printer = class {
    print = (self: &Printer, a: i64, b: i64) { ... }
    print = (self: &Printer, x: i64, y: i64) { ... }  // Error: duplicate method signature
}
```

# Alternatives

1. **Extension Methods (Kotlin-style)**: Allow defining methods outside the class definition. Rejected for simplicity - methods should be colocated with data.

2. **Traits/Interfaces First**: Could add interfaces before classes. Rejected because classes provide more immediate value and interfaces can be added later.

3. **Prototype-based (JavaScript-style)**: More flexible but less predictable. Class-based OOP is more familiar and has simpler codegen.

4. **Separate `impl` blocks (Rust-style)**: Methods defined outside the class body. Rejected because it separates data from behavior unnecessarily for a simple language.

5. **Implicit `self` (Kotlin/Swift-style)**: Could inject `self` implicitly without declaring it in the parameter list. Rejected in favor of explicit `self` parameter like Python/Rust - this makes ownership semantics clear (`self: &T` vs `self: &&T` vs `self: *T`) and explicitly documents the method's relationship to its receiver.

6. **`this` instead of `self`**: Both are common. `self` chosen for consistency with Rust and Python.

7. **Constructor Overloading / `init` Method**: Could have a special `init` method called after field assignment. Rejected because static factory methods provide the same functionality with explicit, self-documenting names, no hidden behavior, and no additional language concepts. This mirrors Rust's approach where `Type { ... }` is raw construction and `Type::new(...)` handles any logic.

# Testing

- **Lexer tests**: Token recognition for `class`, `self`, `object`
- **Parser tests**:
  - Class declaration parsing
  - Object declaration parsing
  - Method declaration parsing with explicit `self` parameter
  - Method call expression parsing
  - Self expression parsing
  - Mixed fields and methods
  - Self parameter type parsing (`self: &T`, `self: &&T`, `self: *T`)
- **Semantic tests**:
  - Class type registration
  - Object type registration
  - Method lookup and validation
  - Method overload resolution
  - Self type checking
  - Method argument validation
  - Static vs instance method enforcement
  - Error detection for invalid method calls
  - Object validation (no fields, no self parameters)
  - Object instantiation error detection
  - Ambiguous overload detection
  - Self parameter validation (SEP 1):
    - `self: &T` prevents field mutation
    - `self: &&T` allows mutation of `var` fields
    - `self: *T` marks instance as moved after call
- **Ownership tests** (SEP 1 integration):
  - `self: *T` methods consume the instance
  - Use-after-move detection for consumed instances
  - Class fields with `*T` types
  - Static factories returning `*ClassName`
  - Passing class instances to `&T` and `*T` parameters
- **Codegen tests**:
  - Method code generation
  - Method call code generation
  - Self access code generation
  - Constructor code generation
  - Ownership transfer in method calls
- **E2E tests** in `_examples/slang/classes/`:
  - Basic class with methods
  - Direct field construction
  - Static factory methods (including derived fields)
  - Method chaining
  - Self field access and modification
  - Static methods
  - Singleton objects
  - Method overloading
  - Classes with multiple methods
  - Heap-allocated class instances (SEP 1)
  - Methods with `self: &T`, `self: &&T`, `self: *T` (SEP 1)
  - Classes containing owned pointers (SEP 1)

# Code Examples

## Example 1: Basic Class with Methods

Demonstrates a simple class with fields and an instance method.

```slang
Counter = class {
    var count: i64

    increment = (self: &&Counter) {
        self.count = self.count + 1
    }

    getCount = (self: &Counter) -> i64 {
        self.count
    }
}

main = () {
    val c = Counter{ 0 }
    c.increment()
    c.increment()
    c.increment()
    print(c.getCount())               // prints: 3
}
```

## Example 2: Method with Parameters

Shows a method that takes parameters and returns a value.

```slang
Calculator = class {
    val base: i64

    add = (self: &Calculator, n: i64) -> i64 {
        self.base + n
    }

    multiply = (self: &Calculator, n: i64) -> i64 {
        self.base * n
    }
}

main = () {
    val calc = Calculator{ 10 }
    print(calc.add(5))                // prints: 15
    print(calc.multiply(3))           // prints: 30
}
```

## Example 3: Factory Method with Derived Fields

Demonstrates using a static factory method to compute derived fields during construction.

```slang
Point = class {
    val x: i64
    val y: i64
    val magnitude: i64    // derived field

    // Factory method computes magnitude
    new = (x: i64, y: i64) -> Point {
        Point{ x, y, x * x + y * y }
    }

    getMagnitude = (self: &Point) -> i64 {
        self.magnitude
    }
}

main = () {
    val p = Point.new(3, 4)
    print(p.getMagnitude())           // prints: 25 (3*3 + 4*4)

    // Direct construction requires all fields
    val p2 = Point{ 3, 4, 25 }
    print(p2.getMagnitude())          // prints: 25
}
```

## Example 4: Mutating Methods

Shows methods that modify fields through mutable borrow.

```slang
Builder = class {
    var value: i64

    add = (self: &&Builder, n: i64) {
        self.value = self.value + n
    }

    multiply = (self: &&Builder, n: i64) {
        self.value = self.value * n
    }

    getValue = (self: &Builder) -> i64 {
        self.value
    }
}

main = () {
    val builder = Builder{ 0 }
    builder.add(5)
    builder.multiply(2)
    builder.add(10)
    print(builder.getValue())         // prints: 20 ((0+5)*2+10)
}
```

Note: Methods cannot return borrowed references (`&T` or `&&T`). For mutation patterns, use void-returning methods with sequential calls as shown above.

## Example 5: Singleton Objects

Demonstrates singleton objects with static methods only.

```slang
Math = object {
    max = (a: i64, b: i64) -> i64 {
        when {
            a > b -> a
            else -> b
        }
    }

    min = (a: i64, b: i64) -> i64 {
        when {
            a < b -> a
            else -> b
        }
    }

    abs = (n: i64) -> i64 {
        when {
            n < 0 -> 0 - n
            else -> n
        }
    }
}

main = () {
    print(Math.max(10, 20))           // prints: 20
    print(Math.min(10, 20))           // prints: 10
    print(Math.abs(0 - 42))           // prints: 42
    // val m = Math{}                 // Error: cannot instantiate object
}
```

## Example 6: Class with Multiple Fields and Methods

Shows a more complete class example.

```slang
Rectangle = class {
    val width: i64
    val height: i64

    area = (self: &Rectangle) -> i64 {
        self.width * self.height
    }

    perimeter = (self: &Rectangle) -> i64 {
        2 * (self.width + self.height)
    }

    isSquare = (self: &Rectangle) -> bool {
        self.width == self.height
    }
}

main = () {
    val rect = Rectangle{ 10, 5 }
    print(rect.area())                // prints: 50
    print(rect.perimeter())           // prints: 30
    print(rect.isSquare())            // prints: false

    val square = Rectangle{ 7, 7 }
    print(square.isSquare())          // prints: true
}
```

## Example 7: Self Field Modification

Shows modifying fields through self in methods.

```slang
BankAccount = class {
    var balance: i64

    deposit = (self: &&BankAccount, amount: i64) {
        self.balance = self.balance + amount
    }

    withdraw = (self: &&BankAccount, amount: i64) -> bool {
        when {
            amount > self.balance -> false
            else -> {
                self.balance = self.balance - amount
                true
            }
        }
    }

    getBalance = (self: &BankAccount) -> i64 {
        self.balance
    }
}

main = () {
    val account = BankAccount{ 100 }
    account.deposit(50)
    print(account.getBalance())       // prints: 150

    val success = account.withdraw(30)
    print(success)                    // prints: true
    print(account.getBalance())       // prints: 120

    val failed = account.withdraw(200)
    print(failed)                     // prints: false
    print(account.getBalance())       // prints: 120 (unchanged)
}
```

## Example 8: Passing Class Instances to Functions

Shows classes working with regular functions.

```slang
Point = class {
    val x: i64
    val y: i64

    distanceFromOrigin = (self: &Point) -> i64 {
        self.x * self.x + self.y * self.y
    }
}

// Regular function that takes a class instance
printPoint = (p: &Point) {
    print(p.x)
    print(p.y)
}

// Function that calls a method on the instance
printDistance = (p: &Point) {
    print(p.distanceFromOrigin())
}

main = () {
    val p = Point{ 3, 4 }
    printPoint(p)                     // prints: 3, 4
    printDistance(p)                  // prints: 25
}
```

## Example 9: Nested Class Usage

Shows classes containing other class instances.

```slang
Point = class {
    val x: i64
    val y: i64

    toString = (self: &Point) -> string {
        "point"
    }
}

Line = class {
    val start: Point
    val end: Point

    length = (self: &Line) -> i64 {
        val dx = self.end.x - self.start.x
        val dy = self.end.y - self.start.y
        dx * dx + dy * dy
    }
}

main = () {
    val p1 = Point{ 0, 0 }
    val p2 = Point{ 3, 4 }
    val line = Line{ p1, p2 }
    print(line.length())              // prints: 25 (squared length)
}
```

## Example 10: Static Factory Methods (Multiple Constructors)

Demonstrates the recommended pattern for providing multiple ways to construct instances.

```slang
Color = class {
    val r: i64
    val g: i64
    val b: i64

    // Static factory methods provide named "constructors"
    // (no self parameter = static method)
    black = () -> Color {
        Color{ 0, 0, 0 }
    }

    white = () -> Color {
        Color{ 255, 255, 255 }
    }

    red = () -> Color {
        Color{ 255, 0, 0 }
    }

    gray = (level: i64) -> Color {
        Color{ level, level, level }
    }

    fromHex = (hex: i64) -> Color {
        // Extract RGB components from hex value
        val r = (hex / 65536) % 256
        val g = (hex / 256) % 256
        val b = hex % 256
        Color{ r, g, b }
    }

    brightness = (self: &Color) -> i64 {
        (self.r + self.g + self.b) / 3
    }
}

main = () {
    val c1 = Color{ 100, 150, 200 }  // direct construction
    val c2 = Color.black()            // factory: (0, 0, 0)
    val c3 = Color.white()            // factory: (255, 255, 255)
    val c4 = Color.gray(128)          // factory: (128, 128, 128)

    print(c1.brightness())            // prints: 150
    print(c2.brightness())            // prints: 0
    print(c3.brightness())            // prints: 255
    print(c4.brightness())            // prints: 128
}
```

## Example 11: Error Cases

Shows compile-time errors for invalid class usage.

```slang
// Error: self outside method
// main = () {
//     print(self.x)                  // Error: 'self' can only be used inside a method
// }

// Error: wrong argument count
Counter = class {
    var count: i64
    add = (self: &&Counter, n: i64) {
        self.count = self.count + n
    }
}

// main = () {
//     val c = Counter{ 0 }
//     c.add()                        // Error: method 'add' expects 1 argument, got 0
// }

// Error: calling static method on instance (or vice versa)
Math = class {
    double = (n: i64) -> i64 {
        n * 2
    }
}

// main = () {
//     val m = Math{}
//     m.double(5)                    // Error: cannot call static method on instance
// }
```

## Example 12: Heap-Allocated Class with Receiver Modes (SEP 1)

Shows method receiver types for ownership control.

```slang
Point = class {
    var x: i64
    var y: i64

    // Static factory returning owned pointer
    new = (x: i64, y: i64) -> *Point {
        Heap.new(Point{ x, y })
    }

    // Immutable borrow - read only
    magnitude = (self: &Point) -> i64 {
        self.x * self.x + self.y * self.y
    }

    // Mutable borrow - can modify fields
    scale = (self: &&Point, factor: i64) {
        self.x = self.x * factor
        self.y = self.y * factor
    }

    // Takes ownership - consumes the instance
    consume = (self: *Point) -> i64 {
        self.x + self.y
    }   // self freed here
}

main = () {
    var p = Point.new(3, 4)

    print(p.magnitude())              // prints: 25 (borrows immutably)
    p.scale(2)                        // borrows mutably
    print(p.x)                        // prints: 6

    val sum = p.consume()             // p consumed
    print(sum)                        // prints: 18 (6 + 12)
    // print(p.x)                     // Error: p was moved
}
```

## Example 13: Class with Owned Pointer Fields (SEP 1)

Shows classes containing owned pointers.

```slang
Node = class {
    val value: i64
    var next: *Node?

    new = (value: i64) -> *Node {
        Heap.new(Node{ value, null })
    }

    // Mutable borrow to modify next
    setNext = (self: &&Node, node: *Node) {
        self.next = node              // ownership transferred to field
    }

    // Immutable borrow to traverse
    sum = (self: &Node) -> i64 {
        when {
            self.next == null -> self.value
            else -> self.value + (self.next?.sum() ?: 0)
        }
    }
}

main = () {
    var n1 = Node.new(10)
    var n2 = Node.new(20)
    var n3 = Node.new(30)

    n2.setNext(n3)                    // n3 moved into n2
    n1.setNext(n2)                    // n2 moved into n1

    print(n1.sum())                   // prints: 60
}                                     // n1 freed, recursively frees chain
```

## Example 14: Passing Classes to Functions (SEP 1)

Shows class instances with function ownership types.

```slang
Point = class {
    var x: i64
    var y: i64

    new = (x: i64, y: i64) -> *Point {
        Heap.new(Point{ x, y })
    }
}

// Immutable borrow
printPoint = (p: &Point) {
    print(p.x)
    print(p.y)
}

// Mutable borrow
doublePoint = (p: &&Point) {
    p.x = p.x * 2
    p.y = p.y * 2
}

// Takes ownership
consumePoint = (p: *Point) {
    print(p.x + p.y)
}   // p freed here

main = () {
    var p = Point.new(5, 10)

    printPoint(p)                     // borrows immutably
    doublePoint(p)                    // borrows mutably
    print(p.x)                        // prints: 10

    consumePoint(p)                   // ownership transferred
    // printPoint(p)                  // Error: p was moved
}
```

# Implementation Order

1. **Lexer** - Add `class`, `self` tokens
2. **AST** - Add `ClassDecl`, `MethodDecl`, `MethodCallExpr`, `SelfExpr`
3. **Parser** - Parse class declarations, methods with explicit `self` parameter, method calls
4. **Types** - Add `ClassType`, `MethodInfo`
5. **Semantic** - Class registration, method analysis, self handling
6. **Ownership** (SEP 1) - Self parameter type validation, move tracking for `self: *T`
7. **Typed AST** - Add typed class nodes
8. **Codegen** - Method generation, method call generation, ownership transfer
9. **E2E Tests** - Integration tests including ownership scenarios

# Files to Modify

| File | Changes |
|------|---------|
| `compiler/lexer/lexer.go` | Add `TokenTypeClass`, `TokenTypeSelf`, `TokenTypeObject` |
| `compiler/ast/ast.go` | Add `ClassDecl`, `ObjectDecl`, `MethodDecl`, `MethodCallExpr`, `SelfExpr` |
| `compiler/parser/parser.go` | Parse class/object declarations, methods with `self` parameter, method calls |
| `compiler/semantic/types.go` | Add `ClassType`, `ObjectType`, `MethodInfo` |
| `compiler/semantic/typed_ast.go` | Add typed class and object nodes |
| `compiler/semantic/analyzer.go` | Class/object registration, method analysis, ownership tracking |
| `compiler/codegen/typed_codegen.go` | Method and method call codegen with ownership transfer |

# Design Decisions

These questions have been resolved:

## 1. Explicit Self Parameter ✅

**Decision:** Methods require an explicit `self` parameter with type annotation.

```slang
Counter = class {
    var count: i64
    increment = (self: &&Counter) {
        self.count = self.count + 1
    }
}
```

This is more verbose but makes ownership semantics clear and explicit.

## 2. Stack-Allocated Class Instances ✅

**Decision:** Auto-borrow for `&T` and `&&T` receivers. Disallow `*T` on stack values.

```slang
Counter = class {
    var count: i64
    increment = (self: &&Counter) { ... }
    consume = (self: *Counter) { ... }  // takes ownership
}

main = () {
    val c = Counter{ 0 }   // stack-allocated
    c.increment()          // OK: auto-borrows as &&Counter
    // c.consume()         // Error: cannot call consuming method on stack value

    val h = Heap.new(Counter{ 0 })  // heap-allocated
    h.consume()            // OK: h is *Counter
}
```

This is consistent with SEP 1's auto-borrowing for function parameters, while preventing the semantic mismatch of "consuming" a stack value.

## 3. Static vs Instance Methods ✅

**Decision:** Methods are distinguished by the presence of `self` as the first parameter.

- **Instance method:** Has `self` as first parameter, called on instance
- **Static method:** No `self` parameter, called on class name

No `static` keyword is needed - the presence or absence of `self` determines the method type.

```slang
Point = class {
    val x: i64
    val y: i64

    // Instance method - has self as first parameter
    magnitude = (self: &Point) -> i64 {
        self.x * self.x + self.y * self.y
    }

    // Static method - no self parameter
    origin = () -> Point {
        Point{ 0, 0 }
    }
}

main = () {
    val p = Point{ 3, 4 }
    p.magnitude()      // instance method
    Point.origin()     // static method
}
```

## 4. Class vs Struct ✅

**Decision:** Add a separate `class` keyword. Structs remain data-only.

- `struct` = data only (fields, no methods)
- `class` = data + methods (fields and methods)

## 5. Method Chaining ✅

**Decision:** Method chaining is supported via owned pointers (`*T`). Methods can return `*T` to enable chaining, but cannot return borrowed references (`&T` or `&&T`).

```slang
Builder = class {
    var value: i64

    // Takes ownership, returns ownership - enables chaining
    add = (self: *Builder, n: i64) -> *Builder {
        self.value = self.value + n
        self
    }

    multiply = (self: *Builder, n: i64) -> *Builder {
        self.value = self.value * n
        self
    }

    build = (self: *Builder) -> i64 {
        self.value
    }   // self freed here
}

main = () {
    // Owned chaining - ownership flows through
    val result = Heap.new(Builder{ 0 })
        .add(5)
        .multiply(2)
        .add(10)
        .build()    // returns 20, Builder freed

    print(result)   // prints: 20
}
```

**Chaining rules:**
- **Owned chains** (`*T` → `*T`): Ownership flows through; original variable is moved after first call
- **No borrow chains**: Methods cannot return `&T` or `&&T` (see Design Decision #13)
- **Mutation without chaining**: For `&&T` methods, use sequential calls instead of chaining

```slang
// For mutable borrow methods, use sequential calls:
Counter = class {
    var count: i64
    increment = (self: &&Counter) {  // void return
        self.count = self.count + 1
    }
}

main = () {
    val c = Counter{ 0 }
    c.increment()   // sequential
    c.increment()   // calls
    c.increment()
    print(c.count)  // 3 - c still usable
}
```

## 6. Feature Interactions ✅

Classes work naturally with other Slang features:

**Arrays of classes:**
```slang
val points = [Point{ 0, 0 }, Point{ 1, 1 }]
points[0].magnitude()                        // method on array element

val heapPoints: Array<*Point> = [Heap.new(Point{ 0, 0 })]
heapPoints[0].magnitude()                    // method on heap-allocated element
```

**Nullable class pointers:**
```slang
Node = class {
    val value: i64
    var next: *Node?                         // nullable owned pointer
}
```

**Nested classes:**
```slang
Line = class {
    val start: Point                         // embedded class instance
    val end: Point

    length = (self: &Line) -> i64 {
        val dx = self.end.x - self.start.x   // access nested fields
        dx * dx
    }
}
```

## 7. Self Parameter Rules ✅

**Decision:** When present, `self` must be the first parameter.

```slang
// Correct - self is first
add = (self: &Counter, n: i64) -> i64 { ... }

// Error - self must be first
add = (n: i64, self: &Counter) -> i64 { ... }
```

## 8. Method and Field Name Conflicts ✅

**Decision:** A method can have the same name as a field. They are distinguished by call syntax.

```slang
Counter = class {
    val count: i64                              // field
    count = (self: &Counter) -> i64 { self.count }  // method (same name OK)
}

main = () {
    val c = Counter{ 42 }
    print(c.count)     // field access: 42
    print(c.count())   // method call: 42
}
```

## 9. Passing Self to Functions ✅

**Decision:** `self` can be passed to other functions like any other value.

```slang
printPoint = (p: &Point) {
    print(p.x)
    print(p.y)
}

Point = class {
    val x: i64
    val y: i64

    display = (self: &Point) {
        printPoint(self)    // OK - passes self as &Point
    }
}
```

## 10. Class Name Shadowing ✅

**Decision:** Variable names cannot shadow class names (compile error).

```slang
Point = class { val x: i64 }

main = () {
    // val Point = 42       // Error: cannot shadow class name 'Point'
    val p = Point{ 10 }     // OK
}
```

## 11. Recursive Self-Reference ✅

**Decision:** Classes can reference themselves in field types (via pointers).

```slang
Node = class {
    val value: i64
    var next: *Node?    // OK - forward reference to self type
}
```

## 12. Forward References Between Classes ✅

**Decision:** Forward references are allowed for all types via two-pass resolution.

The semantic analyzer performs two passes:
1. **First pass:** Collect all class/struct/object names
2. **Second pass:** Resolve field types and method signatures

This allows mutual references regardless of declaration order:

```slang
// Person references Company before it's defined - OK
Person = class {
    val name: string
    var employer: *Company?
}

// Company references Person - OK
Company = class {
    val name: string
    var ceo: *Person?
}

// Works for embedded types too, not just pointers
Line = class {
    val start: Point    // Point defined later - OK
    val end: Point
}

Point = class {
    val x: i64
    val y: i64
}
```

## 13. Methods Cannot Return Borrowed References ✅

**Decision:** Methods cannot return borrowed references (`&T` or `&&T`). Methods can only return owned values (`T` or `*T`).

This simplifies lifetime analysis - borrowed references have scope constraints that would complicate return value handling. For mutation patterns, use void-returning methods with sequential calls:

```slang
Builder = class {
    var value: i64

    // Correct: void return, mutates self
    add = (self: &&Builder, n: i64) {
        self.value = self.value + n
    }

    // Correct: returns owned value
    getValue = (self: &Builder) -> i64 {
        self.value
    }

    // ERROR: cannot return borrowed reference
    // getRef = (self: &Builder) -> &Builder { self }
}

main = () {
    val b = Builder{ 0 }
    b.add(5)           // sequential mutation calls
    b.add(10)
    print(b.getValue())  // prints: 15
}
```

## 14. `val`/`var` Controls Reassignment Only ✅

**Decision:** The `val`/`var` keywords control whether a binding can be reassigned, not whether the value's contents can be mutated.

- `val p = Point{ 1, 2 }` - cannot reassign `p`, but can mutate `p`'s `var` fields
- `var p = Point{ 1, 2 }` - can reassign `p` and mutate `p`'s `var` fields

This means a `val` binding can still call `&&T` methods that modify `var` fields:

```slang
Point = class {
    var x: i64
    var y: i64

    scale = (self: &&Point, factor: i64) {
        self.x = self.x * factor
        self.y = self.y * factor
    }
}

main = () {
    val p = Point{ 1, 2 }   // val = can't reassign p
    p.scale(2)              // OK - can still mutate p's var fields
    print(p.x)              // prints: 2

    // p = Point{ 3, 4 }    // Error - can't reassign val binding
}
```

This is consistent with SEP 1's ownership model where `val`/`var` controls binding reassignability, while `&T`/`&&T` controls borrow mutability.

## 15. Explicit `self.` Required for Field Access ✅

**Decision:** Inside methods, fields must be accessed with explicit `self.` prefix.

```slang
Point = class {
    val x: i64
    val y: i64

    magnitude = (self: &Point) -> i64 {
        // Correct: explicit self
        self.x * self.x + self.y * self.y

        // Error: x is undefined
        // x * x + y * y
    }
}
```

This avoids ambiguity with local variables and is consistent with Python and Rust.

## 16. Borrow Conflict Rules ✅

**Decision:** Borrowing rules prevent conflicts within expressions.

```slang
Point = class {
    var x: i64

    addTo = (self: &&Point, other: &Point) {
        self.x = self.x + other.x
    }

    swap = (self: &&Point, other: &&Point) {
        val temp = self.x
        self.x = other.x
        other.x = temp
    }
}

main = () {
    val p = Point{ 10 }

    // Error: cannot borrow 'p' as immutable because it is already borrowed as mutable
    p.addTo(p)

    // Error: cannot borrow 'p' as mutable more than once
    p.swap(p)
}
```

## 17. Namespace Rules ✅

**Decision:** Classes, structs, objects, and functions share the same namespace.

```slang
Point = class { val x: i64 }

// Error: type 'Point' is already defined
Point = struct { val y: i64 }

// Error: type 'Point' is already defined
Point = object { }

// Error: 'Point' is already defined as a class
Point = () -> i64 { 42 }
```

However, method names are scoped to their class and can match top-level function names:

```slang
print = (x: i64) { }    // top-level function

Logger = class {
    print = (self: &Logger, x: i64) { }   // OK: method name scoped to class
}
```

## 18. Circular Dependency Detection ✅

**Decision:** Circular dependencies via non-pointer fields are detected during semantic analysis.

```slang
// Error: circular dependency between 'A' and 'B' via non-pointer fields
A = class {
    val b: B
}

B = class {
    val a: A
}

// OK: pointers break the cycle
A = class {
    val b: *B?
}

B = class {
    val a: *A?
}
```

## 19. Empty Classes and Objects ✅

**Decision:** Empty classes and objects are allowed.

```slang
Empty = class { }           // OK: zero-size instance
EmptyUtil = object { }      // OK: valid but useless

main = () {
    val e = Empty{}         // creates zero-size instance
}
```

## 20. Parameter Shadowing ✅

**Decision:** Local variables cannot shadow method parameters.

```slang
Point = class {
    compute = (self: &Point, value: i64) -> i64 {
        // Error: cannot shadow parameter 'value'
        val value = value * 2
        value
    }
}
```

## 21. Consuming Methods on Temporaries ✅

**Decision:** Consuming methods (`self: *T`) cannot be called on temporaries.

```slang
Point = class {
    consume = (self: *Point) -> i64 {
        self.x + self.y
    }
}

main = () {
    // Error: cannot call consuming method 'consume' on temporary;
    // use Heap.new(Point{ 3, 4 }).consume() instead
    val result = Point{ 3, 4 }.consume()
}
```

## 22. Error Messages ✅

Clear error messages for invalid patterns:

| Pattern | Error Message |
|---------|---------------|
| `self` not first parameter | `'self' must be the first parameter` |
| `self` in method body but not in params | `cannot use 'self' in static method 'X'` |
| `self` type not a pointer | `'self' must have pointer type (&X, &&X, or *X), got Y` |
| `self` type references wrong class | `'self' type must reference enclosing class 'X', got 'Y'` |
| Consuming method on stack value | `cannot call consuming method 'X' on stack-allocated value; use Heap.new() for heap allocation` |
| Instance method called as static | `cannot call instance method 'X' without a receiver; use 'instance.X()'` |
| Static method called on instance | `cannot call static method 'X' on instance; use 'ClassName.X()'` |
| Variable shadows class name | `cannot shadow class name 'X' with variable` |
| Method not found | `class 'X' has no method 'Y'` |
| Wrong argument count | `method 'X' expects N arguments, got M` |
| Wrong argument type | `method 'X' parameter 'Y' expects type A, got B` |
| Instantiating an object | `cannot instantiate object 'X'` |
| Field in object | `objects cannot have fields` |
| `self` parameter in object method | `object methods cannot have 'self' parameter` |
| Duplicate method signature | `duplicate method signature for 'X'` |
| No matching overload | `no matching overload for method 'X' with arguments (A, B)` (lists available overloads) |
| Returning borrowed reference | `methods cannot return borrowed references (&T or &&T); return owned value instead` |
| Method call on struct | `cannot call method on struct type 'X'; structs do not have methods` |
| Method call on null | `cannot call method on 'null'; type is unknown` |
| Consuming method on temporary | `cannot call consuming method 'X' on temporary; use Heap.new(...).X() instead` |
| Mutable borrow conflict | `cannot borrow 'X' as immutable because it is already borrowed as mutable` |
| Multiple mutable borrows | `cannot borrow 'X' as mutable more than once` |
| Implicit field access | `undefined variable 'X'` (must use `self.X` for field access in methods) |
| Parameter shadowing | `cannot shadow parameter 'X'` |
| Duplicate type name | `type 'X' is already defined` (classes, structs, objects share namespace) |
| Function shadows type | `'X' is already defined as a class` |
| Circular non-pointer dependency | `circular dependency between 'X' and 'Y' via non-pointer fields` |

# Risks and Limitations

1. **Memory Management**: Classes allocated on heap require allocation strategy. The existing bump allocator from SEP 1 can be used for heap-allocated class instances.

2. **No Inheritance**: Without inheritance, code reuse is limited. This is intentional for the first version - composition can be used instead.

3. **No Visibility**: All fields and methods are public. Private fields can be added later with `private` keyword.

4. **Method Dispatch**: All method calls are statically dispatched (no virtual methods). This is simpler but limits polymorphism.

5. **Self Mutability**: `self` is immutable by default. Mutating fields through `self` works because we're mutating the pointed-to data, not `self` itself.

# Future Enhancements

These are explicitly out of scope but may be added later:

1. **Interfaces**
   ```slang
   Printable = interface {
       toString = (self: &Self) -> string
   }
   ```

2. **Visibility Modifiers**
   ```slang
   BankAccount = class {
       private var balance: i64
       public getBalance = (self: &BankAccount) -> i64 { self.balance }
   }
   ```

3. **Inheritance**
   ```slang
   Animal = class {
       speak = (self: &Animal) { print("...") }
   }
   Dog = class extends Animal {
       speak = (self: &Dog) { print("woof") }
   }
   ```

4. **Generic Classes**
   ```slang
   Box = class<T> {
       val value: T
       get = (self: &Box<T>) -> T { self.value }
   }
   ```
