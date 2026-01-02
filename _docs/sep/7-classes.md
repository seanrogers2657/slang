# Status

DRAFT, 2025-12-31

# Summary/Motivation

Add classes to Slang, extending structs with methods (functions bound to a type). This enables encapsulation of data and behavior together, allowing types to have associated operations without passing the instance explicitly to every function.

# Goals/Non-Goals

- [goal] Class declaration syntax with `class` keyword using assignment-based format
- [goal] Method declarations inside class body with implicit `self` parameter
- [goal] `self` keyword for accessing the current instance within methods
- [goal] Method calls using dot notation (`instance.method()`)
- [goal] Field declarations with `val`/`var` (like structs)
- [goal] Direct field construction via struct-literal syntax (`ClassName{ ... }`)
- [goal] Static methods via `static` modifier
- [non-goal] Inheritance (single or multiple)
- [non-goal] Visibility modifiers (`public`, `private`, `protected`)
- [non-goal] Abstract classes or interfaces
- [non-goal] Operator overloading
- [non-goal] Generic/parameterized classes (future enhancement)
- [non-goal] Properties with custom getters/setters
- [non-goal] Constructor overloading (use static factory methods instead)

# APIs

- `class` - Keyword for declaring a class type with fields and methods.
- `self` - Keyword referencing the current instance within method bodies.
- `static` - Modifier for methods that don't operate on an instance.
- `.method()` - Dot notation for calling methods on instances.
- `ClassName{ ... }` - Direct field construction (same as struct syntax).

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

## Step 1: Lexer Changes

**File:** `compiler/lexer/lexer.go`

Add token support for class syntax:

```go
// New token types
TokenTypeClass   // 'class' keyword
TokenTypeSelf    // 'self' keyword
TokenTypeStatic  // 'static' keyword
```

Add to keywords map:
```go
"class":  TokenTypeClass,
"self":   TokenTypeSelf,
"static": TokenTypeStatic,
```

## Step 2: AST Changes

**File:** `compiler/ast/ast.go`

Add new AST nodes for class declarations:

```go
// ClassDecl represents a class declaration
type ClassDecl struct {
    Name       string
    NamePos    Position
    Fields     []FieldDecl      // Same as struct fields
    Methods    []MethodDecl
    StaticMethods []MethodDecl
}

// MethodDecl represents a method declaration within a class
type MethodDecl struct {
    Name       string
    NamePos    Position
    Params     []ParamDecl      // Does NOT include 'self' (implicit)
    ReturnType string           // Empty for void
    ReturnPos  Position
    Body       *BlockStatement
    IsStatic   bool
}

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
    var staticMethods []ast.MethodDecl

    for p.currentToken.Type != TokenTypeRBrace {
        if p.currentToken.Type == TokenTypeStatic {
            p.advance()
            method := p.parseMethodDecl(true)
            staticMethods = append(staticMethods, method)
        } else if p.currentToken.Type == TokenTypeVal ||
                  p.currentToken.Type == TokenTypeVar {
            field := p.parseFieldDecl()
            fields = append(fields, field)
        } else {
            method := p.parseMethodDecl(false)
            methods = append(methods, method)
        }
    }

    p.expect(TokenTypeRBrace) // '}'

    return &ast.ClassDecl{
        Name: name,
        NamePos: namePos,
        Fields: fields,
        Methods: methods,
        StaticMethods: staticMethods,
    }
}
```

### Parse Method Declaration

Methods use the same assignment syntax as functions:

```go
func (p *Parser) parseMethodDecl(isStatic bool) ast.MethodDecl {
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
        IsStatic:   isStatic,
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
    Name          string
    Fields        []FieldInfo
    Methods       map[string]*MethodInfo
    StaticMethods map[string]*MethodInfo
}

// MethodInfo contains method signature information
type MethodInfo struct {
    Name       string
    ParamTypes []Type
    ReturnType Type
    IsStatic   bool
}

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

    // Parse methods
    methods := make(map[string]*MethodInfo)
    for _, m := range decl.Methods {
        methods[m.Name] = &MethodInfo{
            Name:       m.Name,
            ParamTypes: a.resolveParamTypes(m.Params),
            ReturnType: a.resolveTypeName(m.ReturnType, m.ReturnPos),
            IsStatic:   false,
        }
    }

    // Parse static methods
    staticMethods := make(map[string]*MethodInfo)
    for _, m := range decl.StaticMethods {
        staticMethods[m.Name] = &MethodInfo{
            Name:       m.Name,
            ParamTypes: a.resolveParamTypes(m.Params),
            ReturnType: a.resolveTypeName(m.ReturnType, m.ReturnPos),
            IsStatic:   true,
        }
    }

    a.types[decl.Name] = ClassType{
        Name:          decl.Name,
        Fields:        fields,
        Methods:       methods,
        StaticMethods: staticMethods,
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

    // Add 'self' to scope (immutable reference to class instance)
    if !method.IsStatic {
        a.currentScope.Define("self", VariableInfo{
            Type:    classType,
            Mutable: false,  // self is immutable
        })
    }

    // Add parameters to scope
    for _, param := range method.Params {
        a.currentScope.Define(param.Name, VariableInfo{
            Type:    a.resolveTypeName(param.TypeName, param.TypePos),
            Mutable: false,
        })
    }

    // Analyze body
    typedBody := a.analyzeBlockStatement(method.Body)

    // Check return type
    // ...

    return &TypedMethodDecl{...}
}
```

### Analyze Method Calls

```go
func (a *Analyzer) analyzeMethodCallExpr(expr *ast.MethodCallExpr) TypedExpression {
    // Analyze receiver
    typedObject := a.analyzeExpression(expr.Object)
    objectType := typedObject.ExprType()

    // Check object is a class type
    classType, ok := objectType.(ClassType)
    if !ok {
        a.addError("cannot call method on non-class type %s", objectType)
        return errorExpr(expr)
    }

    // Look up method
    methodInfo, exists := classType.Methods[expr.Method]
    if !exists {
        a.addError("class %s has no method %s", classType.Name, expr.Method)
        return errorExpr(expr)
    }

    // Type check arguments
    typedArgs := a.analyzeArguments(expr.Args, methodInfo.ParamTypes)

    return &TypedMethodCallExpr{
        Type:   methodInfo.ReturnType,
        Object: typedObject,
        Method: expr.Method,
        Args:   typedArgs,
    }
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
    Name          string
    ClassType     ClassType
    Fields        []TypedFieldDecl
    Methods       []*TypedMethodDecl
    StaticMethods []*TypedMethodDecl
}

type TypedMethodDecl struct {
    Name       string
    Params     []TypedParamDecl
    ReturnType Type
    Body       *TypedBlockStatement
    IsStatic   bool
}

type TypedMethodCallExpr struct {
    Type   Type
    Object TypedExpression
    Method string
    Args   []TypedExpression
}

type TypedSelfExpr struct {
    Type Type
    Pos  ast.Position
}

func (e *TypedMethodCallExpr) ExprType() Type { return e.Type }
func (e *TypedSelfExpr) ExprType() Type       { return e.Type }
```

## Step 7: Code Generation

**File:** `compiler/codegen/typed_codegen.go`

### Method Table Layout

Each class has a method table (vtable) stored in the data section:

```
_ClassName_methods:
    .quad _ClassName_method1
    .quad _ClassName_method2
    ...
```

### Instance Layout

Class instances store a pointer to the method table followed by fields:

```
+0:  method_table_ptr (8 bytes)
+8:  field1
+16: field2
...
```

### Generate Method Code

Methods are generated as regular functions with mangled names:

```go
func (g *TypedCodeGenerator) generateMethodDecl(className string, method *TypedMethodDecl) string {
    builder := strings.Builder{}

    // Method label: _ClassName_methodName
    methodLabel := fmt.Sprintf("_%s_%s", className, method.Name)
    builder.WriteString(fmt.Sprintf(".global %s\n", methodLabel))
    builder.WriteString(fmt.Sprintf("%s:\n", methodLabel))

    // Prologue
    builder.WriteString("    stp x29, x30, [sp, #-16]!\n")
    builder.WriteString("    mov x29, sp\n")

    // Allocate space for locals
    // 'self' is passed in x0, params in x1, x2, x3, ...

    // Store self on stack
    builder.WriteString("    str x0, [sp, #-16]!\n")  // self at [sp]

    // Store parameters
    for i, param := range method.Params {
        reg := fmt.Sprintf("x%d", i+1)  // x1, x2, x3...
        offset := (i + 1) * 16
        builder.WriteString(fmt.Sprintf("    str %s, [sp, #-%d]!\n", reg, offset))
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

### Generate Method Call

```go
func (g *TypedCodeGenerator) generateMethodCallExpr(expr *TypedMethodCallExpr, ctx *CodeGenContext) (string, error) {
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

    // Get class type to find method
    classType := expr.Object.ExprType().(ClassType)
    methodLabel := fmt.Sprintf("_%s_%s", classType.Name, expr.Method)

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

    // Calculate instance size
    instanceSize := 8 + len(classType.Fields)*8  // vtable ptr + fields

    // Allocate memory (using mmap or stack allocation)
    builder.WriteString(fmt.Sprintf("    mov x0, #%d\n", instanceSize))
    builder.WriteString("    bl _sl_alloc\n")  // allocation helper
    builder.WriteString("    mov x9, x0\n")    // save instance ptr

    // Store vtable pointer
    builder.WriteString(fmt.Sprintf("    adrp x10, _%s_vtable@PAGE\n", className))
    builder.WriteString(fmt.Sprintf("    add x10, x10, _%s_vtable@PAGEOFF\n", className))
    builder.WriteString("    str x10, [x9]\n")

    // Initialize fields from arguments (direct assignment, no init method)
    for i, arg := range args {
        argCode, _ := g.generateExpr(arg, ctx)
        builder.WriteString(argCode)
        offset := 8 + i*8  // skip vtable ptr
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
    static origin = () -> Point {
        Point{ 0, 0 }
    }

    static fromX = (x: i64) -> Point {
        Point{ x, 0 }
    }

    static fromY = (y: i64) -> Point {
        Point{ 0, y }
    }

    static diagonal = (n: i64) -> Point {
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
    static new = (r: i64) -> Circle {
        Circle{ r, r * r * 3 }    // calculate area inline
    }

    static unit = () -> Circle {
        Circle.new(1)
    }

    static withDiameter = (d: i64) -> Circle {
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
    add = (n: i64) {
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
    increment = () {
        self.count = self.count + 1
    }
}
main = () {
    Counter.increment()             // Error: cannot call instance method 'increment' without receiver
}
```

# Alternatives

1. **Extension Methods (Kotlin-style)**: Allow defining methods outside the class definition. Rejected for simplicity - methods should be colocated with data.

2. **Traits/Interfaces First**: Could add interfaces before classes. Rejected because classes provide more immediate value and interfaces can be added later.

3. **Prototype-based (JavaScript-style)**: More flexible but less predictable. Class-based OOP is more familiar and has simpler codegen.

4. **Separate `impl` blocks (Rust-style)**: Methods defined outside the class body. Rejected because it separates data from behavior unnecessarily for a simple language.

5. **Implicit `self` (Python-style explicit in params)**: Could require `self` in parameter list. Rejected because implicit `self` is cleaner and matches Kotlin/Swift.

6. **`this` instead of `self`**: Both are common. `self` chosen for consistency with Rust and Python.

7. **Constructor Overloading / `init` Method**: Could have a special `init` method called after field assignment. Rejected because static factory methods provide the same functionality with explicit, self-documenting names, no hidden behavior, and no additional language concepts. This mirrors Rust's approach where `Type { ... }` is raw construction and `Type::new(...)` handles any logic.

# Testing

- **Lexer tests**: Token recognition for `class`, `self`, `static`
- **Parser tests**:
  - Class declaration parsing
  - Method declaration parsing
  - Method call expression parsing
  - Self expression parsing
  - Mixed fields and methods
- **Semantic tests**:
  - Class type registration
  - Method lookup and validation
  - Self type checking
  - Method argument validation
  - Static vs instance method enforcement
  - Error detection for invalid method calls
- **Codegen tests**:
  - Method code generation
  - Method call code generation
  - Self access code generation
  - Constructor code generation
- **E2E tests** in `_examples/slang/classes/`:
  - Basic class with methods
  - Direct field construction
  - Static factory methods (including derived fields)
  - Method chaining
  - Self field access and modification
  - Static methods
  - Classes with multiple methods

# Code Examples

## Example 1: Basic Class with Methods

Demonstrates a simple class with fields and an instance method.

```slang
Counter = class {
    var count: i64

    increment = () {
        self.count = self.count + 1
    }

    getCount = () -> i64 {
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

    add = (n: i64) -> i64 {
        self.base + n
    }

    multiply = (n: i64) -> i64 {
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
    static new = (x: i64, y: i64) -> Point {
        Point{ x, y, x * x + y * y }
    }

    getMagnitude = () -> i64 {
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

## Example 4: Method Chaining

Shows methods that return `self` for fluent interfaces.

```slang
StringBuilder = class {
    var value: i64

    add = (n: i64) -> StringBuilder {
        self.value = self.value + n
        self
    }

    multiply = (n: i64) -> StringBuilder {
        self.value = self.value * n
        self
    }

    getValue = () -> i64 {
        self.value
    }
}

main = () {
    val builder = StringBuilder{ 0 }
    val result = builder.add(5).multiply(2).add(10).getValue()
    print(result)                     // prints: 20 ((0+5)*2+10)
}
```

## Example 5: Static Methods

Demonstrates static methods that don't operate on an instance.

```slang
Math = class {
    static max = (a: i64, b: i64) -> i64 {
        when {
            a > b -> a
            else -> b
        }
    }

    static min = (a: i64, b: i64) -> i64 {
        when {
            a < b -> a
            else -> b
        }
    }

    static abs = (n: i64) -> i64 {
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
}
```

## Example 6: Class with Multiple Fields and Methods

Shows a more complete class example.

```slang
Rectangle = class {
    val width: i64
    val height: i64

    area = () -> i64 {
        self.width * self.height
    }

    perimeter = () -> i64 {
        2 * (self.width + self.height)
    }

    isSquare = () -> bool {
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

    deposit = (amount: i64) {
        self.balance = self.balance + amount
    }

    withdraw = (amount: i64) -> bool {
        when {
            amount > self.balance -> false
            else -> {
                self.balance = self.balance - amount
                true
            }
        }
    }

    getBalance = () -> i64 {
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

    distanceFromOrigin = () -> i64 {
        self.x * self.x + self.y * self.y
    }
}

// Regular function that takes a class instance
printPoint = (p: Point) {
    print(p.x)
    print(p.y)
}

// Function that calls a method on the instance
printDistance = (p: Point) {
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

    toString = () -> string {
        "point"
    }
}

Line = class {
    val start: Point
    val end: Point

    length = () -> i64 {
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
    static black = () -> Color {
        Color{ 0, 0, 0 }
    }

    static white = () -> Color {
        Color{ 255, 255, 255 }
    }

    static red = () -> Color {
        Color{ 255, 0, 0 }
    }

    static gray = (level: i64) -> Color {
        Color{ level, level, level }
    }

    static fromHex = (hex: i64) -> Color {
        // Extract RGB components from hex value
        val r = (hex / 65536) % 256
        val g = (hex / 256) % 256
        val b = hex % 256
        Color{ r, g, b }
    }

    brightness = () -> i64 {
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
    add = (n: i64) {
        self.count = self.count + n
    }
}

// main = () {
//     val c = Counter{ 0 }
//     c.add()                        // Error: method 'add' expects 1 argument, got 0
// }

// Error: calling static method on instance (or vice versa)
Math = class {
    static double = (n: i64) -> i64 {
        n * 2
    }
}

// main = () {
//     val m = Math{}
//     m.double(5)                    // Error: cannot call static method on instance
// }
```

# Implementation Order

1. **Lexer** - Add `class`, `self`, `static` tokens
2. **AST** - Add `ClassDecl`, `MethodDecl`, `MethodCallExpr`, `SelfExpr`
3. **Parser** - Parse class declarations, methods, method calls
4. **Types** - Add `ClassType`, `MethodInfo`
5. **Semantic** - Class registration, method analysis, self handling
6. **Typed AST** - Add typed class nodes
7. **Codegen** - Method generation, method call generation
8. **E2E Tests** - Integration tests

# Files to Modify

| File | Changes |
|------|---------|
| `compiler/lexer/lexer.go` | Add `TokenTypeClass`, `TokenTypeSelf`, `TokenTypeStatic` |
| `compiler/ast/ast.go` | Add `ClassDecl`, `MethodDecl`, `MethodCallExpr`, `SelfExpr` |
| `compiler/parser/parser.go` | Parse class declarations, methods, method calls |
| `compiler/semantic/types.go` | Add `ClassType`, `MethodInfo` |
| `compiler/semantic/typed_ast.go` | Add typed class nodes |
| `compiler/semantic/analyzer.go` | Class registration, method analysis |
| `compiler/codegen/typed_codegen.go` | Method and method call codegen |

# Risks and Limitations

1. **Memory Management**: Classes allocated on heap require allocation strategy. Initial implementation may use simple bump allocator or stack allocation for simplicity.

2. **No Inheritance**: Without inheritance, code reuse is limited. This is intentional for the first version - composition can be used instead.

3. **No Visibility**: All fields and methods are public. Private fields can be added later with `private` keyword.

4. **Method Dispatch**: All method calls are statically dispatched (no virtual methods). This is simpler but limits polymorphism.

5. **Self Mutability**: `self` is immutable by default. Mutating fields through `self` works because we're mutating the pointed-to data, not `self` itself.

# Future Enhancements

These are explicitly out of scope but may be added later:

1. **Interfaces**
   ```slang
   Printable = interface {
       toString = () -> string
   }
   ```

2. **Visibility Modifiers**
   ```slang
   BankAccount = class {
       private var balance: i64
       public getBalance = () -> i64 { self.balance }
   }
   ```

3. **Inheritance**
   ```slang
   Animal = class {
       speak = () { print("...") }
   }
   Dog = class extends Animal {
       speak = () { print("woof") }
   }
   ```

4. **Generic Classes**
   ```slang
   Box = class<T> {
       val value: T
       get = () -> T { self.value }
   }
   ```
