# Status

DRAFT, 2025-12-31

# Summary/Motivation

Add heap allocation through pointer types to Slang, enabling dynamic memory allocation and data structures like linked lists and trees. This introduces `ptr<T>` types with `ptr::new(value)` for allocation and `.deref()` for dereferencing, with auto-dereference convenience for field access and array indexing.

# Goals/Non-Goals

- [goal] Heap allocation via `ptr::new(value)` returning `ptr<T>`
- [goal] Explicit dereference with `.deref()` method
- [goal] Auto-dereference for field access (`p.field`) and array indexing (`p[i]`) like Go
- [goal] Type inference for pointer types
- [goal] Support for nested pointers and pointers within structs
- [non-goal] Memory deallocation (`ptr::free()`)
- [non-goal] Null pointers (`ptr::null<T>()`)
- [non-goal] Pointer mutation/assignment (`p.set(newValue)`)
- [non-goal] Address-of operator (`&variable`)
- [non-goal] Pointer arithmetic
- [non-goal] Garbage collection or reference counting

# APIs

- `ptr<T>` - New generic pointer type representing a heap-allocated value of type T.
- `ptr::new(value)` - Allocates memory on the heap, stores the value, and returns a `ptr<T>`.
- `.deref()` - Method on `ptr<T>` that returns the pointed-to value of type T.
- Auto-dereference on field access - `ptr<Struct>.field` automatically dereferences to access struct fields.
- Auto-dereference on indexing - `ptr<Array<T>>[i]` automatically dereferences to access array elements.

# Description

## Step 1: Lexer Changes

Add token support for pointer syntax:
- Add `ptr` keyword token
- Handle `::` operator for `ptr::new`

## Step 2: Parser Changes

Extend the parser to handle pointer expressions and types:
- Parse `ptr<T>` type syntax in type annotations
- Parse `ptr::new(expr)` as allocation expressions
- Parse `.deref()` as method call expressions

## Step 3: Type System Changes

Add pointer type to the semantic analyzer:
- Add `PointerType` struct wrapping an inner type T
- Type check `ptr::new(expr)` - infer T from argument type, return `ptr<T>`
- Type check `.deref()` on `ptr<T>` - return inner type T
- Error on `.deref()` called on non-pointer types

## Step 4: Auto-Dereference Rules

Implement convenience auto-dereference in the semantic analyzer:
- Field access on `ptr<Struct>` auto-dereferences to access struct fields (Example 2)
- Index access on `ptr<Array<T>>` auto-dereferences to access array elements (Example 3)
- Chain through multiple pointer levels (Example 4)
- Error on field access for `ptr<primitive>` or index access for `ptr<Struct>`

## Step 5: Code Generation

Generate ARM64 assembly for pointer operations:
- Heap allocation using `mmap` syscall (or `brk`)
- Store value at allocated address
- Load from pointer address for `.deref()`
- Handle auto-dereference in field/index access codegen

## Error Handling

```slang
val x = 42
x.deref()                                 // Error: .deref() on non-pointer

val p = ptr::new(42)
print(p.x)                                // Error: ptr<i64> has no fields

val p = ptr::new(Point{ 1, 2 })
print(p[0])                               // Error: ptr<Point> not indexable
```

## Quick Reference

| Expression | Type of `p` | Result Type | Notes |
|------------|-------------|-------------|-------|
| `p.deref()` | `ptr<T>` | `T` | Works for any T |
| `p.x` | `ptr<Struct>` | field type | Auto-deref |
| `p[i]` | `ptr<Array<T>>` | `T` | Auto-deref |
| `p.field.deref()` | `ptr<S>` where S has `field: ptr<T>` | `T` | Chained |
| `p.deref().deref()` | `ptr<ptr<T>>` | `T` | Nested ptr |

# Alternatives

1. **Rust-style references (`&T`, `&mut T`)**: More complex borrow checker required. Rejected for MVP simplicity.

2. **C-style pointers (`*T`, `*p`)**: Less explicit, potential confusion with multiplication. The `ptr<T>` syntax is clearer and more consistent with generic types.

3. **Implicit dereference everywhere**: Would hide pointer semantics too much. The `.deref()` requirement makes pointer operations explicit while auto-deref provides convenience where it's unambiguous.

4. **Manual memory management from start**: Would complicate MVP. Starting with allocation-only allows proving the design before adding deallocation complexity.

# Testing

- **Lexer tests**: Token recognition for `ptr`, `::`, `new`, `deref`
- **Parser tests**: `ptr<T>` type parsing, `ptr::new(expr)` expressions, `.deref()` method calls
- **Semantic tests**: Type inference, auto-dereference rules, error detection
- **Codegen tests**: Correct assembly for allocation and dereference
- **E2E tests**: Full programs using pointers with expected outputs
  - Basic allocation and dereference
  - Pointers to structs with field access
  - Pointers to arrays with indexing
  - Nested pointers
  - Error cases (compile-time errors)

# Code Examples

## Example 1: Basic Allocation and Dereference

Demonstrates basic pointer allocation with `ptr::new` and explicit dereferencing with `.deref()`.

```slang
main = () {
    val p = ptr::new(42)
    print(p.deref())                      // prints: 42

    val sum = p.deref() + 8
    print(sum)                            // prints: 50
}
```

## Example 2: Struct with Auto-Dereference

Shows auto-dereference for field access on a pointer to a struct.

```slang
Point = struct {
    val x: i64
    val y: i64
}

main = () {
    val p = ptr::new(Point{ 10, 20 })
    print(p.x)                            // prints: 10 (auto-deref)
    print(p.y)                            // prints: 20 (auto-deref)
    print(p.x + p.y)                      // prints: 30
}
```

## Example 3: Array with Auto-Dereference

Shows auto-dereference for index access on a pointer to an array.

```slang
main = () {
    val arr = ptr::new([1, 2, 3, 4, 5])
    print(arr[0])                         // prints: 1 (auto-deref)
    print(arr[2])                         // prints: 3
    print(arr[0] + arr[4])                // prints: 6
}
```

## Example 4: Linked List with Chained Auto-Dereference

Demonstrates chained auto-dereference through multiple pointer levels in a linked list.

```slang
Node = struct {
    val value: i64
    val next: ptr<Node>
}

main = () {
    val n3 = ptr::new(Node{ 30, ??? })    // null handling TBD
    val n2 = ptr::new(Node{ 20, n3 })
    val n1 = ptr::new(Node{ 10, n2 })

    print(n1.value)                       // prints: 10
    print(n1.next.value)                  // prints: 20
    print(n1.next.next.value)             // prints: 30
}
```

## Example 5: Function Returning Pointer

Shows a function that allocates and returns a pointer.

```slang
createPoint = (x: i64, y: i64) -> ptr<Point> {
    ptr::new(Point{ x, y })
}

main = () {
    val p = createPoint(5, 10)
    print(p.x + p.y)                      // prints: 15
}
```
