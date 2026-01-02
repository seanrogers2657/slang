# Status

DRAFT, 2025-12-31

# Summary/Motivation

Add interfaces to Slang, enabling polymorphism through contracts that classes can implement. Interfaces define method signatures that implementing classes must provide, allowing code to work with any type that satisfies the interface contract.

# Goals/Non-Goals

- [goal] Interface declaration syntax with `interface` keyword using assignment-based format
- [goal] Method signature declarations (no implementations)
- [goal] `implements` clause for classes to declare interface conformance
- [goal] Multiple interface implementation per class
- [goal] Interface types for polymorphic variables and parameters
- [goal] Dynamic dispatch for interface-typed method calls
- [goal] Compile-time verification that classes implement all interface methods
- [non-goal] Static methods in interfaces
- [non-goal] Default method implementations
- [non-goal] Interface inheritance (interface extending interface)
- [non-goal] Generic/parameterized interfaces
- [non-goal] Structural typing (automatic interface satisfaction without `implements`)
- [non-goal] Associated types or `Self` type in interfaces

# APIs

- `interface` - Keyword for declaring an interface with method signatures.
- `implements` - Clause on class declarations to specify implemented interfaces.
- Interface types - Interfaces can be used as types for variables, parameters, and return values.

# Description

## Relationship to Classes

Interfaces complement classes by defining contracts:

| Feature | Class | Interface |
|---------|-------|-----------|
| Fields | Yes | No |
| Method implementations | Yes | No (signatures only) |
| Static methods | Yes | No |
| Instantiation | Yes (`ClassName{ ... }`) | No |
| Used as type | Yes | Yes |
| Method dispatch | Static | Dynamic (vtable) |

## Step 1: Lexer Changes

**File:** `compiler/lexer/lexer.go`

Add token support for interface syntax:

```go
// New token types
TokenTypeInterface   // 'interface' keyword
TokenTypeImplements  // 'implements' keyword
```

Add to keywords map:
```go
"interface":  TokenTypeInterface,
"implements": TokenTypeImplements,
```

## Step 2: AST Changes

**File:** `compiler/ast/ast.go`

Add new AST nodes for interface declarations:

```go
// InterfaceDecl represents an interface declaration
type InterfaceDecl struct {
    Name       string
    NamePos    Position
    Methods    []MethodSignature
}

// MethodSignature represents a method signature in an interface (no body)
type MethodSignature struct {
    Name       string
    NamePos    Position
    Params     []ParamDecl
    ReturnType string
    ReturnPos  Position
}
```

Update ClassDecl to include implements:

```go
// ClassDecl represents a class declaration
type ClassDecl struct {
    Name          string
    NamePos       Position
    Implements    []string      // interface names
    ImplementsPos []Position
    Fields        []FieldDecl
    Methods       []MethodDecl
    StaticMethods []MethodDecl
}
```

## Step 3: Parser Changes

**File:** `compiler/parser/parser.go`

### Parse Interface Declaration

At top-level, when encountering `<Name> = interface {`:

```go
func (p *Parser) parseInterfaceDecl() *ast.InterfaceDecl {
    // Current token is identifier (interface name)
    name := p.currentToken.Value
    namePos := p.currentToken.Position

    p.advance() // consume name
    p.expect(TokenTypeAssign)     // '='
    p.expect(TokenTypeInterface)  // 'interface'
    p.expect(TokenTypeLBrace)     // '{'

    var methods []ast.MethodSignature

    for p.currentToken.Type != TokenTypeRBrace {
        method := p.parseMethodSignature()
        methods = append(methods, method)
    }

    p.expect(TokenTypeRBrace) // '}'

    return &ast.InterfaceDecl{
        Name:    name,
        NamePos: namePos,
        Methods: methods,
    }
}
```

### Parse Method Signature

Method signatures use same syntax as methods but without body:

```go
func (p *Parser) parseMethodSignature() ast.MethodSignature {
    // Pattern: methodName = (params) -> ReturnType
    //      or: methodName = (params)  (void return)

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

    // No body - just the signature

    return ast.MethodSignature{
        Name:       name,
        NamePos:    namePos,
        Params:     params,
        ReturnType: returnType,
        ReturnPos:  returnPos,
    }
}
```

### Parse Class with Implements

Update class parsing to handle `implements`:

```go
func (p *Parser) parseClassDecl() *ast.ClassDecl {
    name := p.currentToken.Value
    namePos := p.currentToken.Position

    p.advance() // consume name
    p.expect(TokenTypeAssign) // '='
    p.expect(TokenTypeClass)  // 'class'

    // Check for 'implements'
    var implements []string
    var implementsPos []Position
    if p.currentToken.Type == TokenTypeImplements {
        p.advance() // consume 'implements'

        // Parse comma-separated interface names
        for {
            implements = append(implements, p.currentToken.Value)
            implementsPos = append(implementsPos, p.currentToken.Position)
            p.advance()

            if p.currentToken.Type != TokenTypeComma {
                break
            }
            p.advance() // consume ','
        }
    }

    p.expect(TokenTypeLBrace) // '{'

    // ... rest of class parsing (fields, methods, static methods)
}
```

## Step 4: Type System Changes

**File:** `compiler/semantic/types.go`

Add interface type representation:

```go
// InterfaceType represents an interface with method signatures
type InterfaceType struct {
    Name    string
    Methods map[string]*MethodSignatureInfo
}

// MethodSignatureInfo contains method signature information
type MethodSignatureInfo struct {
    Name       string
    ParamTypes []Type
    ReturnType Type
}

func (t InterfaceType) String() string {
    return t.Name
}

func (t InterfaceType) Equals(other Type) bool {
    o, ok := other.(InterfaceType)
    if !ok {
        return false
    }
    return t.Name == o.Name
}
```

Add method to check if a class implements an interface:

```go
func (c ClassType) Implements(iface InterfaceType) bool {
    for methodName, sig := range iface.Methods {
        classMethod, exists := c.Methods[methodName]
        if !exists {
            return false
        }
        if !methodSignaturesMatch(classMethod, sig) {
            return false
        }
    }
    return true
}

func methodSignaturesMatch(method *MethodInfo, sig *MethodSignatureInfo) bool {
    if len(method.ParamTypes) != len(sig.ParamTypes) {
        return false
    }
    for i, pt := range method.ParamTypes {
        if !pt.Equals(sig.ParamTypes[i]) {
            return false
        }
    }
    return method.ReturnType.Equals(sig.ReturnType)
}
```

## Step 5: Semantic Analysis

**File:** `compiler/semantic/analyzer.go`

### Register Interface Types

During declaration phase, register interface types:

```go
func (a *Analyzer) registerInterfaceType(decl *ast.InterfaceDecl) {
    // Check for duplicate type name
    if _, exists := a.types[decl.Name]; exists {
        a.addError("type '%s' is already declared", decl.Name)
        return
    }

    // Parse method signatures
    methods := make(map[string]*MethodSignatureInfo)
    for _, m := range decl.Methods {
        // Check for duplicate method names
        if _, exists := methods[m.Name]; exists {
            a.addError("duplicate method '%s' in interface '%s'", m.Name, decl.Name)
            continue
        }

        methods[m.Name] = &MethodSignatureInfo{
            Name:       m.Name,
            ParamTypes: a.resolveParamTypes(m.Params),
            ReturnType: a.resolveTypeName(m.ReturnType, m.ReturnPos),
        }
    }

    a.types[decl.Name] = InterfaceType{
        Name:    decl.Name,
        Methods: methods,
    }
}
```

### Validate Class Implements

When analyzing a class declaration, verify it implements all declared interfaces:

```go
func (a *Analyzer) validateClassImplements(classDecl *ast.ClassDecl, classType ClassType) {
    for i, ifaceName := range classDecl.Implements {
        // Look up interface type
        ifaceType, exists := a.types[ifaceName]
        if !exists {
            a.addError("unknown interface '%s'", ifaceName)
            continue
        }

        iface, ok := ifaceType.(InterfaceType)
        if !ok {
            a.addError("'%s' is not an interface", ifaceName)
            continue
        }

        // Check each interface method is implemented
        for methodName, sig := range iface.Methods {
            classMethod, exists := classType.Methods[methodName]
            if !exists {
                a.addError("class '%s' does not implement method '%s' from interface '%s'",
                    classDecl.Name, methodName, ifaceName)
                continue
            }

            if !methodSignaturesMatch(classMethod, sig) {
                a.addError("method '%s' in class '%s' has wrong signature for interface '%s'",
                    methodName, classDecl.Name, ifaceName)
            }
        }

        // Record that this class implements this interface
        classType.ImplementedInterfaces = append(classType.ImplementedInterfaces, iface)
    }
}
```

### Type Compatibility with Interfaces

Update type compatibility checking:

```go
func (a *Analyzer) isAssignableTo(fromType, toType Type) bool {
    // Same type
    if fromType.Equals(toType) {
        return true
    }

    // Class assignable to interface it implements
    if classType, isClass := fromType.(ClassType); isClass {
        if ifaceType, isIface := toType.(InterfaceType); isIface {
            return classType.Implements(ifaceType)
        }
    }

    return false
}
```

## Step 6: Typed AST

**File:** `compiler/semantic/typed_ast.go`

Add typed nodes:

```go
type TypedInterfaceDecl struct {
    Name          string
    InterfaceType InterfaceType
    Methods       []TypedMethodSignature
}

type TypedMethodSignature struct {
    Name       string
    ParamTypes []Type
    ReturnType Type
}
```

Update ClassType to track implemented interfaces:

```go
type ClassType struct {
    Name                  string
    Fields                []FieldInfo
    Methods               map[string]*MethodInfo
    StaticMethods         map[string]*MethodInfo
    ImplementedInterfaces []InterfaceType
}
```

## Step 7: Code Generation

**File:** `compiler/codegen/typed_codegen.go`

### Interface Vtable Layout

Each class generates a vtable for each interface it implements:

```asm
// For: Circle = class implements Drawable { ... }
_Circle_Drawable_vtable:
    .quad _Circle_draw        // offset 0: draw method
    .quad _Circle_getBounds   // offset 8: getBounds method
```

### Dynamic Dispatch

When calling a method on an interface-typed value, use vtable lookup:

```go
func (g *TypedCodeGenerator) generateInterfaceMethodCall(
    expr *TypedMethodCallExpr,
    ifaceType InterfaceType,
    ctx *CodeGenContext,
) (string, error) {
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

    // Load interface vtable pointer from object
    // Object layout: [class_vtable, iface1_vtable, iface2_vtable, ..., fields...]
    ifaceVtableOffset := g.getInterfaceVtableOffset(expr.Object.ExprType(), ifaceType)
    builder.WriteString(fmt.Sprintf("    ldr x9, [x0, #%d]\n", ifaceVtableOffset))

    // Load method address from interface vtable
    methodOffset := g.getMethodOffsetInInterface(ifaceType, expr.Method)
    builder.WriteString(fmt.Sprintf("    ldr x10, [x9, #%d]\n", methodOffset))

    // Call method through pointer
    builder.WriteString("    blr x10\n")

    // Result is in x0, move to x2 for consistency
    builder.WriteString("    mov x2, x0\n")

    return builder.String(), nil
}
```

### Instance Layout with Interfaces

Class instances include vtable pointers for each implemented interface:

```
+0:  class_vtable_ptr
+8:  Drawable_vtable_ptr     (if implements Drawable)
+16: Serializable_vtable_ptr (if implements Serializable)
+24: field1
+32: field2
...
```

```go
func (g *TypedCodeGenerator) generateClassConstruction(
    className string,
    args []TypedExpression,
    ctx *CodeGenContext,
) (string, error) {
    builder := strings.Builder{}
    classType := g.types[className].(ClassType)

    // Calculate instance size
    numVtables := 1 + len(classType.ImplementedInterfaces)
    instanceSize := numVtables*8 + len(classType.Fields)*8

    // Allocate memory
    builder.WriteString(fmt.Sprintf("    mov x0, #%d\n", instanceSize))
    builder.WriteString("    bl _sl_alloc\n")
    builder.WriteString("    mov x9, x0\n")

    // Store class vtable pointer
    builder.WriteString(fmt.Sprintf("    adrp x10, _%s_vtable@PAGE\n", className))
    builder.WriteString(fmt.Sprintf("    add x10, x10, _%s_vtable@PAGEOFF\n", className))
    builder.WriteString("    str x10, [x9]\n")

    // Store interface vtable pointers
    for i, iface := range classType.ImplementedInterfaces {
        offset := (i + 1) * 8
        vtableLabel := fmt.Sprintf("_%s_%s_vtable", className, iface.Name)
        builder.WriteString(fmt.Sprintf("    adrp x10, %s@PAGE\n", vtableLabel))
        builder.WriteString(fmt.Sprintf("    add x10, x10, %s@PAGEOFF\n", vtableLabel))
        builder.WriteString(fmt.Sprintf("    str x10, [x9, #%d]\n", offset))
    }

    // Initialize fields
    fieldOffset := numVtables * 8
    for i, arg := range args {
        argCode, _ := g.generateExpr(arg, ctx)
        builder.WriteString(argCode)
        builder.WriteString(fmt.Sprintf("    str x2, [x9, #%d]\n", fieldOffset+i*8))
    }

    builder.WriteString("    mov x2, x9\n")
    return builder.String(), nil
}
```

## Error Handling

```slang
// Unknown interface
Circle = class implements Unknown {    // Error: unknown interface 'Unknown'
    val radius: i64
}

// Implementing non-interface type
Circle = class implements Point {      // Error: 'Point' is not an interface
    val radius: i64
}

// Missing interface method
Drawable = interface {
    draw = () -> void
}

Circle = class implements Drawable {
    val radius: i64
    // Error: class 'Circle' does not implement method 'draw' from interface 'Drawable'
}

// Wrong method signature
Drawable = interface {
    draw = () -> void
}

Circle = class implements Drawable {
    val radius: i64

    draw = (color: i64) -> void {      // Error: method 'draw' has wrong signature
        print(color)
    }
}

// Cannot instantiate interface
Drawable = interface {
    draw = () -> void
}

main = () {
    val d = Drawable{}                 // Error: cannot instantiate interface 'Drawable'
}

// Type mismatch - class doesn't implement interface
Drawable = interface {
    draw = () -> void
}

Point = class {
    val x: i64
    val y: i64
}

main = () {
    val p = Point{ 10, 20 }
    val d: Drawable = p                // Error: 'Point' does not implement 'Drawable'
}
```

# Alternatives

1. **Structural Typing (Go-style)**: Classes automatically satisfy interfaces if they have matching methods. Rejected because explicit `implements` is clearer and catches errors earlier.

2. **Traits with Default Implementations (Rust-style)**: Allow interfaces to provide default method bodies. Rejected for simplicity - can be added later if needed.

3. **Interface Inheritance**: Allow interfaces to extend other interfaces. Rejected for simplicity - a class can implement multiple interfaces instead.

4. **`Self` Type in Interfaces**: Allow methods to reference the implementing type. Rejected for initial simplicity - requires more complex type system support.

5. **Static Methods in Interfaces**: Allow interfaces to require static methods. Rejected because static methods are called on types, not instances, which complicates the interface model.

6. **`protocol` instead of `interface`**: Swift uses `protocol`. Rejected because `interface` is more widely recognized from Java/Go/TypeScript.

# Testing

- **Lexer tests**: Token recognition for `interface`, `implements`
- **Parser tests**:
  - Interface declaration parsing
  - Method signature parsing (no body)
  - Class with `implements` clause
  - Multiple interface implementation
- **Semantic tests**:
  - Interface type registration
  - Class implements validation
  - Missing method detection
  - Wrong signature detection
  - Type compatibility (class to interface)
  - Error detection for invalid implementations
- **Codegen tests**:
  - Interface vtable generation
  - Dynamic dispatch code
  - Instance layout with interface vtables
- **E2E tests** in `_examples/slang/interfaces/`:
  - Basic interface and implementation
  - Multiple interface implementation
  - Interface as parameter type
  - Interface as return type
  - Polymorphic collections
  - Factory methods returning interface types

# Code Examples

## Example 1: Basic Interface

Demonstrates a simple interface with one implementing class.

```slang
Drawable = interface {
    draw = () -> void
}

Circle = class implements Drawable {
    val x: i64
    val y: i64
    val radius: i64

    draw = () {
        print("Drawing circle")
    }

    static new = (x: i64, y: i64, r: i64) -> Circle {
        Circle{ x, y, r }
    }
}

main = () {
    val c = Circle.new(10, 20, 5)
    c.draw()                          // prints: Drawing circle
}
```

## Example 2: Interface as Parameter Type

Shows polymorphism by passing different types to a function expecting an interface.

```slang
Printable = interface {
    toString = () -> string
}

Point = class implements Printable {
    val x: i64
    val y: i64

    toString = () -> string {
        "Point"
    }

    static new = (x: i64, y: i64) -> Point {
        Point{ x, y }
    }
}

Color = class implements Printable {
    val r: i64
    val g: i64
    val b: i64

    toString = () -> string {
        "Color"
    }
}

// Function accepting interface type
printItem = (item: Printable) {
    print(item.toString())
}

main = () {
    val p = Point.new(10, 20)
    val c = Color{ 255, 128, 0 }

    printItem(p)                      // prints: Point
    printItem(c)                      // prints: Color
}
```

## Example 3: Multiple Interface Implementation

Shows a class implementing multiple interfaces.

```slang
Drawable = interface {
    draw = () -> void
}

Movable = interface {
    move = (dx: i64, dy: i64) -> void
}

Sprite = class implements Drawable, Movable {
    var x: i64
    var y: i64
    val image: string

    draw = () {
        print(self.image)
    }

    move = (dx: i64, dy: i64) {
        self.x = self.x + dx
        self.y = self.y + dy
    }

    static new = (x: i64, y: i64, image: string) -> Sprite {
        Sprite{ x, y, image }
    }
}

drawAll = (items: Array<Drawable>) {
    for item in items {
        item.draw()
    }
}

moveAll = (items: Array<Movable>, dx: i64, dy: i64) {
    for item in items {
        item.move(dx, dy)
    }
}

main = () {
    val s1 = Sprite.new(0, 0, "player")
    val s2 = Sprite.new(100, 100, "enemy")

    // Can use as either interface type
    drawAll([s1, s2])
    moveAll([s1, s2], 10, 5)
}
```

## Example 4: Interface as Return Type

Shows a factory function returning an interface type.

```slang
Shape = interface {
    area = () -> i64
}

Circle = class implements Shape {
    val radius: i64

    area = () -> i64 {
        self.radius * self.radius * 3
    }
}

Square = class implements Shape {
    val side: i64

    area = () -> i64 {
        self.side * self.side
    }
}

Rectangle = class implements Shape {
    val width: i64
    val height: i64

    area = () -> i64 {
        self.width * self.height
    }
}

// Factory function returning interface type
createShape = (shapeType: i64, size: i64) -> Shape {
    when {
        shapeType == 1 -> Circle{ size }
        shapeType == 2 -> Square{ size }
        else -> Rectangle{ size, size * 2 }
    }
}

main = () {
    val s1 = createShape(1, 5)        // Circle
    val s2 = createShape(2, 4)        // Square
    val s3 = createShape(3, 3)        // Rectangle

    print(s1.area())                  // prints: 75 (5*5*3)
    print(s2.area())                  // prints: 16 (4*4)
    print(s3.area())                  // prints: 18 (3*6)
}
```

## Example 5: Interface with Multiple Methods

Shows an interface with several method signatures.

```slang
Collection = interface {
    size = () -> i64
    isEmpty = () -> bool
    contains = (value: i64) -> bool
}

IntList = class implements Collection {
    val items: Array<i64>

    size = () -> i64 {
        len(self.items)
    }

    isEmpty = () -> bool {
        len(self.items) == 0
    }

    contains = (value: i64) -> bool {
        var found = false
        for item in self.items {
            when {
                item == value -> { found = true }
                else -> {}
            }
        }
        found
    }

    static new = (items: Array<i64>) -> IntList {
        IntList{ items }
    }

    static empty = () -> IntList {
        IntList{ [] }
    }
}

printCollectionInfo = (c: Collection) {
    print(c.size())
    print(c.isEmpty())
}

main = () {
    val list = IntList.new([1, 2, 3, 4, 5])
    val empty = IntList.empty()

    printCollectionInfo(list)         // prints: 5, false
    printCollectionInfo(empty)        // prints: 0, true

    print(list.contains(3))           // prints: true
    print(list.contains(10))          // prints: false
}
```

## Example 6: Polymorphic Variable

Shows storing different implementing types in an interface-typed variable.

```slang
Animal = interface {
    speak = () -> void
    name = () -> string
}

Dog = class implements Animal {
    val dogName: string

    speak = () {
        print("Woof!")
    }

    name = () -> string {
        self.dogName
    }
}

Cat = class implements Animal {
    val catName: string

    speak = () {
        print("Meow!")
    }

    name = () -> string {
        self.catName
    }
}

Bird = class implements Animal {
    val birdName: string

    speak = () {
        print("Tweet!")
    }

    name = () -> string {
        self.birdName
    }
}

main = () {
    // Interface-typed variable can hold any implementor
    var pet: Animal = Dog{ "Rex" }
    pet.speak()                       // prints: Woof!
    print(pet.name())                 // prints: Rex

    pet = Cat{ "Whiskers" }
    pet.speak()                       // prints: Meow!
    print(pet.name())                 // prints: Whiskers

    pet = Bird{ "Tweety" }
    pet.speak()                       // prints: Tweet!
    print(pet.name())                 // prints: Tweety
}
```

## Example 7: Interface with Static Factory Pattern

Shows combining interfaces with the static factory method pattern from classes.

```slang
Serializable = interface {
    serialize = () -> string
    getType = () -> string
}

User = class implements Serializable {
    val id: i64
    val name: string

    serialize = () -> string {
        self.name
    }

    getType = () -> string {
        "User"
    }

    static new = (id: i64, name: string) -> User {
        User{ id, name }
    }

    static guest = () -> User {
        User{ 0, "Guest" }
    }
}

Product = class implements Serializable {
    val sku: string
    val price: i64

    serialize = () -> string {
        self.sku
    }

    getType = () -> string {
        "Product"
    }

    static new = (sku: string, price: i64) -> Product {
        Product{ sku, price }
    }
}

saveToLog = (item: Serializable) {
    print(item.getType())
    print(item.serialize())
}

main = () {
    val user = User.new(1, "Alice")
    val guest = User.guest()
    val product = Product.new("ABC123", 999)

    saveToLog(user)                   // prints: User, Alice
    saveToLog(guest)                  // prints: User, Guest
    saveToLog(product)                // prints: Product, ABC123
}
```

## Example 8: Error Cases

Shows compile-time errors for invalid interface usage.

```slang
// Error: missing method implementation
Drawable = interface {
    draw = () -> void
}

// Circle = class implements Drawable {
//     val radius: i64
//     // Error: class 'Circle' does not implement method 'draw' from interface 'Drawable'
// }

// Error: wrong signature
// Circle = class implements Drawable {
//     val radius: i64
//     draw = (x: i64) -> void {       // Error: wrong signature for 'draw'
//         print(x)
//     }
// }

// Error: cannot instantiate interface
// main = () {
//     val d = Drawable{}              // Error: cannot instantiate interface
// }

// Error: type doesn't implement interface
Point = class {
    val x: i64
    val y: i64
}

// main = () {
//     val d: Drawable = Point{ 1, 2 } // Error: 'Point' does not implement 'Drawable'
// }
```

# Implementation Order

1. **Lexer** - Add `interface`, `implements` tokens
2. **AST** - Add `InterfaceDecl`, `MethodSignature`, update `ClassDecl`
3. **Parser** - Parse interface declarations, `implements` clause
4. **Types** - Add `InterfaceType`, `MethodSignatureInfo`
5. **Semantic** - Interface registration, implements validation, type compatibility
6. **Typed AST** - Add typed interface nodes
7. **Codegen** - Interface vtables, dynamic dispatch
8. **E2E Tests** - Integration tests

# Files to Modify

| File | Changes |
|------|---------|
| `compiler/lexer/lexer.go` | Add `TokenTypeInterface`, `TokenTypeImplements` |
| `compiler/ast/ast.go` | Add `InterfaceDecl`, `MethodSignature`, update `ClassDecl` |
| `compiler/parser/parser.go` | Parse interface declarations, `implements` clause |
| `compiler/semantic/types.go` | Add `InterfaceType`, `MethodSignatureInfo` |
| `compiler/semantic/typed_ast.go` | Add typed interface nodes |
| `compiler/semantic/analyzer.go` | Interface registration, implements validation |
| `compiler/codegen/typed_codegen.go` | Interface vtables, dynamic dispatch |

# Risks and Limitations

1. **Performance**: Dynamic dispatch through vtables is slower than static dispatch. For performance-critical code, use concrete class types instead of interface types.

2. **Memory Overhead**: Each implemented interface adds 8 bytes to instance size for the vtable pointer.

3. **No Structural Typing**: Classes must explicitly declare `implements`. This is intentional for clarity but means existing classes can't satisfy new interfaces without modification.

4. **No Generics**: Without generic interfaces, some patterns (like `Comparable<T>`) aren't possible. This can be addressed when generics are added.

5. **No Default Methods**: All interface methods must be implemented. This means shared behavior requires duplication or helper functions.

# Future Enhancements

These are explicitly out of scope but may be added later:

1. **Default Method Implementations**
   ```slang
   Printable = interface {
       toString = () -> string

       // Default implementation
       print = () {
           print(self.toString())
       }
   }
   ```

2. **Interface Inheritance**
   ```slang
   Drawable = interface {
       draw = () -> void
   }

   AdvancedDrawable = interface extends Drawable {
       drawWithColor = (color: i64) -> void
   }
   ```

3. **`Self` Type**
   ```slang
   Comparable = interface {
       compareTo = (other: Self) -> i64
   }
   ```

4. **Generic Interfaces**
   ```slang
   Iterator = interface<T> {
       next = () -> T?
       hasNext = () -> bool
   }
   ```

5. **Static Interface Methods**
   ```slang
   Defaultable = interface {
       static default = () -> Self
   }
   ```
