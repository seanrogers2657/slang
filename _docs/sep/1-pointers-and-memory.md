# Status

DRAFT, 2026-01-03

# Summary/Motivation

Add heap allocation through pointer types to Slang, enabling dynamic memory allocation and data structures like linked lists and trees. This introduces two pointer types:

- **`Own<T>`** - Owned pointer. You control the lifetime; memory is freed when the owner goes out of scope.
- **`Ref<T>`** - Borrowed reference. Temporary access to someone else's data; no ownership.

Mutability is controlled by `val`/`var`, not by the pointer type:

- **`val`** - Immutable. Cannot reassign or mutate through the pointer.
- **`var`** - Mutable. Can reassign and mutate `var` fields through the pointer.

This keeps ownership and mutability as orthogonal concepts, reusing keywords users already understand.

`Heap` is a built-in allocator type with a `new` method that returns `Own<T>`. This design allows for future extensibility - custom allocators can implement the same interface.

The ownership model is simple:
- **Assignment moves** - `val q = p` transfers ownership, making `p` invalid
- **Function parameters borrow or take ownership** - `Ref<T>` borrows, `Own<T>` takes ownership

Slang already has nullable types (`T?`) and arrays (`Array<T>`). This SEP builds on those foundations - nullable pointers (`Own<T>?`) provide a natural way to represent optional references (e.g., linked list terminators).

# Goals/Non-Goals

## Allocation
- [goal] `Heap` built-in allocator type with `new(value)` method
- [goal] `Heap.new(value)` returns `Own<T>` (owned pointer)
- [goal] Type inference for pointer types

## Pointer Types
- [goal] `Own<T>` - owned pointer, controls lifetime of pointed-to memory
- [goal] `Ref<T>` - borrowed reference, temporary access without ownership
- [goal] Automatic borrowing from `Own<T>` to `Ref<T>` based on context (no explicit `.borrow()`)
- [goal] Auto-dereference for field access (`p.field`) and array indexing (`p[i]`)
- [goal] Nullable pointers via existing `T?` syntax (`Own<T>?` with `null`)

## Mutability
- [goal] `val`/`var` controls mutability orthogonally from ownership
- [goal] `val p: Own<T>` - owned, immutable (can't reassign or mutate through p)
- [goal] `var p: Own<T>` - owned, mutable (can reassign and mutate var fields)
- [goal] `p: Ref<T>` - borrowed, immutable (read-only access)
- [goal] `var p: Ref<T>` - borrowed, mutable (can mutate var fields)

## Ownership & Memory Safety
- [goal] Single ownership model - each allocation has exactly one owner
- [goal] `Ref<T>` parameters borrow (caller keeps ownership)
- [goal] `Own<T>` parameters take ownership (caller loses access)
- [goal] Assignment moves ownership (`val q = p` invalidates `p`)
- [goal] Automatic deallocation when owner goes out of scope
- [goal] Auto-free on reassignment (`var p = ...; p = ...` frees old value)
- [goal] Deep copy with `.copy()` for independent copies
- [goal] Compile-time enforcement of all ownership rules (no runtime cost)

## Allocation Failure
- [goal] `Heap.new()` panics on allocation failure (out of memory)
- [non-goal] Fallible allocation (`Heap.tryNew()`) - future work

## Non-Goals
- [non-goal] Explicit `Heap.free()` - ownership handles deallocation automatically
- [non-goal] Uninitialized allocation (`Heap.alloc<T>()`) - future work
- [non-goal] Fallible allocation (`Heap.tryNew() -> Own<T>?`) - future work
- [non-goal] Custom allocator interface - future work, but design anticipates it
- [non-goal] Address-of operator (`&variable`) - can only get pointers via `Heap.new`
- [non-goal] Pointer arithmetic
- [non-goal] Garbage collection or reference counting
- [non-goal] Explicit lifetime annotations - ownership rules avoid need for these
- [non-goal] Partial moves - cannot move individual struct fields

# APIs

## Allocator

- `Heap` - Built-in allocator type for heap memory allocation.
- `Heap.new(value)` - Allocates memory on the heap, stores the value, and returns `Own<T>`. Type T is inferred from the value. The value can be:
  - A literal: `Heap.new(Point{ 1, 2 })`
  - A stack-allocated variable: `Heap.new(p)` (moves `p` to heap)
  - An owned pointer: `Heap.new(ownedPtr)` (creates `Own<Own<T>>` - a pointer to a pointer)
- **Allocation failure:** `Heap.new()` panics if allocation fails (out of memory). This matches Go, Swift, and Kotlin behavior. For fallible allocation, see Future Work (`Heap.tryNew`).

## Pointer Types

- `Own<T>` - Owned pointer. You control the lifetime; memory freed when owner goes out of scope.
- `Ref<T>` - Borrowed reference. Temporary access without ownership.
- `Own<T>?` - Nullable owned pointer (can be `null` or a valid pointer).
- `.copy()` - Create a deep copy, returning a new independent `Own<T>`.

## Automatic Borrowing

Borrowing happens automatically based on context - no explicit `.borrow()` method needed:

| Context | Source | Result |
|---------|--------|--------|
| Pass `Own<T>` to param `Ref<T>` | `foo(p)` | Auto-borrow (immutable) |
| Pass `Own<T>` to param `var Ref<T>` | `bar(p)` | Auto-borrow (mutable) |
| Method call with `self: Ref<T>` | `p.method()` | Auto-borrow based on receiver |
| Field access on `Own<T>` | `p.field` | Auto-dereference |

## Array Indexing

Array indexing behavior depends on element type:

- **Primitive elements** (`Array<i64>`, etc.): `arr[i]` returns a copy of the value
- **Owned pointer elements** (`Array<Own<T>>`): `arr[i]` returns `Ref<T>` (borrows the element)
- **Mutability inherited**: If the array is `var`, indexing returns `var Ref<T>` for owned elements

```slang
// Primitive array - indexing returns copy
val nums = Heap.new([1, 2, 3])
val x = nums[0]               // x: i64 (copy)

// Owned pointer array - indexing borrows
val points: Own<Array<Own<Point>>> = Heap.new([...])
val p = points[0]             // p: Ref<Point> (immutable borrow)
// points[0] still owns the element

// Mutable array - indexing returns mutable borrow
var mutablePoints: Own<Array<Own<Point>>> = Heap.new([...])
val q = mutablePoints[0]      // q: var Ref<Point> (mutable borrow)
q.x = 100                     // OK: can mutate through var Ref

// To actually remove/replace elements, use methods:
val removed = points.remove(0)    // Returns Own<Point>, shifts elements
val old = points.set(0, newPoint) // Replaces element, returns old one
```

**Rationale:** Moving elements out of arrays via indexing would leave "holes" and require complex tracking. Borrowing is always safe.

## Field Access Through Ref

When accessing an `Own<T>` field through a `Ref`, the result is automatically borrowed:

- Through `Ref<Container>`: field `Own<T>` becomes `Ref<T>`
- Through `var Ref<Container>`: field `Own<T>` becomes `var Ref<T>`

```slang
Container = struct {
    val data: Own<Point>
}

// Immutable access
readData = (c: Ref<Container>) {
    val p = c.data            // p: Ref<Point> (auto-borrow)
    print(p.x)                // OK: read access
    // p.x = 10               // Error: p is not var
}

// Mutable access
mutateData = (var c: Ref<Container>) {
    val p = c.data            // p: var Ref<Point> (auto-borrow, inherits mutability)
    p.x = 10                  // OK: can mutate through var Ref
}
```

**Chained access:** Mutability propagates through the chain.

```slang
Outer = struct { val inner: Own<Inner> }
Inner = struct { val point: Own<Point> }

deepAccess = (var o: Ref<Outer>) {
    val p = o.inner.point     // p: var Ref<Point>
    // o is var Ref<Outer>
    // o.inner is var Ref<Inner> (auto-borrow, inherits var)
    // o.inner.point is var Ref<Point> (auto-borrow, inherits var)
    p.x = 100                 // OK
}
```

**Rationale:** You cannot move out of borrowed data. Auto-borrowing is the only safe option.

## Mutability

Mutability is controlled by `val`/`var` on the binding, not by the pointer type:

| Declaration | Can Reassign | Can Mutate Fields |
|-------------|--------------|-------------------|
| `val p: Own<T>` | No | No |
| `var p: Own<T>` | Yes | Yes (if field is `var`) |
| `p: Ref<T>` | — | No |
| `var p: Ref<T>` | — | Yes (if field is `var`) |

## Type Usage

- `Own<T>` - Can be used anywhere: variables, parameters, return types, struct fields. T must be a non-nullable type.
- `Ref<T>` - Can be used as function parameters. Cannot be stored in struct fields or returned from functions (would dangle). T must be a non-nullable type.
- `Own<T>?` - Nullable owned pointer. The pointer may be null; if non-null, it points to a valid T.
- `Ref<T>?` - Nullable borrowed reference. Can be used as function parameter for optional borrowed data.
- **Recursive types** - Structs can reference themselves via `Own<Self>?` fields (must be nullable to allow a base case).
- **No `Own<T?>`** - Pointers to nullable types are not supported. Use `Own<T>?` instead.

```slang
// ✅ Valid pointer types
val p: Own<Point> = Heap.new(Point{ 1, 2 })
val q: Own<Point>? = null
val r: Own<Array<Own<Point>>> = Heap.new([...])

// ❌ Invalid: T? inside Own
val bad: Own<Point?> = ...             // Error: Own<T> requires non-nullable T
```

```slang
// Recursive type example
Node = struct {
    val value: i64
    var next: Own<Node>?               // nullable self-reference
}

// Optional borrowed data
maybeUse = (p: Ref<Point>?) {
    if (p != null) {
        print(p.x)
    }
}

main = () {
    val p = Heap.new(Point{ 1, 2 })
    maybeUse(p)      // OK: auto-borrow
    maybeUse(null)   // OK: pass null
}
```

## Copyability

Types are either **copyable** (can be duplicated) or **move-only** (must transfer ownership):

- **Copyable types:** Primitives (`i64`, `bool`, `string`) and structs containing only copyable fields
- **Move-only types:** `Own<T>` and any struct containing `Own<T>` fields

```slang
// Copyable - all fields are primitives
Point = struct { val x: i64; val y: i64 }
val p1 = Point{ 1, 2 }
val p2 = p1            // Copy - both valid

// Move-only - contains Own<T>
Container = struct { val data: Own<Point> }
val c1 = Container{ Heap.new(Point{ 1, 2 }) }
val c2 = c1            // Move - c1 is invalid

// To copy move-only types, use .copy()
val c3 = c1.copy()     // Deep copy - both valid (if c1 wasn't moved)
```

This affects array indexing: `arr[i]` returns a copy for copyable element types, but `Ref<T>` for move-only element types.

## Auto-Dereference

- Field access - `p.field` automatically dereferences to access struct fields.
- Index access - `p[i]` automatically dereferences to access array elements.
- Safe navigation - `?.` works with nullable pointers just like other nullable types.
- **Error on non-nullable:** Using `?.` on a non-nullable pointer is a compile error.

```slang
val p = Heap.new(Point{ 1, 2 })       // Own<Point>, not nullable
print(p?.x)                            // Error: safe navigation on non-nullable type

val q: Own<Point>? = maybeGet()
print(q?.x)                            // OK: q is nullable
```

## Comparison

- `p == q` for pointers is **identity comparison** (same address), not value comparison.
- For value comparison, compare fields directly: `p.x == q.x && p.y == q.y`.
- Nullable pointers can be compared to `null`: `p == null`.
- `Own<T>` can be compared with `Ref<T>` - both are identity (address) comparison.

```slang
val p = Heap.new(Point{ 1, 2 })
val q = Heap.new(Point{ 1, 2 })
val r = p.copy()

print(p == q)                         // false: different allocations
print(p == r)                         // false: copy is separate allocation
print(p.x == q.x && p.y == q.y)       // true: same field values

val n: Own<Point>? = null
print(n == null)                      // true

// Own<T> vs Ref<T> comparison
compare = (r: Ref<Point>) -> bool {
    r == p                            // OK: identity comparison
}
print(compare(p))                     // true: same address
print(compare(q))                     // false: different address
```

## Implicit Conversions

- `Own<T>` → `Ref<T>` - Automatic when passing to function expecting `Ref<T>` parameter.
- `Own<T>?` → `Ref<T>` - **Not allowed.** Must unwrap first via null check or smart cast.

```slang
foo = (p: Ref<Point>) { print(p.x) }

main = () {
    val p: Own<Point>? = maybeGet()

    // foo(p)                         // Error: cannot auto-borrow nullable pointer

    if (p != null) {
        foo(p)                         // OK: p is smart cast to Own<Point>
    }
}
```

# Ownership Model

Slang uses a simple ownership model that provides memory safety without garbage collection or lifetime annotations.

**Two concepts to remember:**
1. **Ownership** - `Own<T>` means you control the lifetime; `Ref<T>` means you're borrowing
2. **Mutability** - `val` means immutable; `var` means mutable

## Core Concepts

### Single Ownership

Every `Own<T>` has exactly one owner. When the owner goes out of scope, the memory is automatically freed.

```slang
main = () {
    val p = Heap.new(Point{ 1, 2 })    // p: Own<Point>, main owns it
    print(p.x)
}                                       // p automatically freed here
```

### Move on Assignment

Assigning an owned pointer to a new variable **moves** ownership. The original variable becomes invalid.

```slang
main = () {
    val p = Heap.new(Point{ 1, 2 })
    val q = p                           // ownership moves to q

    print(q.x)                          // OK: q owns the memory
    // print(p.x)                       // Error: p was moved
}
```

### Borrowing with Ref<T>

`Ref<T>` parameters borrow - the caller keeps ownership.

```slang
// Ref<T> parameter = immutable borrow (read-only)
printPoint = (p: Ref<Point>) {
    print(p.x)
    print(p.y)
    // p.x = 100                        // Error: p is not var
}

// var Ref<T> parameter = mutable borrow (read-write)
scalePoint = (var p: Ref<Point>, factor: i64) {
    p.x = p.x * factor                  // OK: p is var
    p.y = p.y * factor
}

main = () {
    var p = Heap.new(Point{ 1, 2 })

    printPoint(p)                       // borrows as Ref<Point> (immutable)
    printPoint(p)                       // can borrow again

    scalePoint(p, 10)                   // borrows as var Ref<Point> (mutable)
    print(p.x)                          // prints: 10

    print(p.x)                          // OK: main still owns p
}
```

### Ownership Transfer with Own<T>

`Own<T>` parameters take ownership - the caller loses access.

```slang
// Own<T> parameter = takes ownership
consume = (p: Own<Point>) {
    print(p.x)
}                                       // p freed here - we own it

main = () {
    val a = Heap.new(Point{ 10, 20 })
    consume(a)                          // ownership transferred
    // print(a.x)                       // Error: a was moved
}
```

### Stack to Heap Promotion

Stack-allocated values can be moved to the heap with `Heap.new()`.

```slang
main = () {
    val p = Point{ 1, 2 }               // stack-allocated
    val h = Heap.new(p)                 // moves p to heap, h: Own<Point>

    // print(p.x)                       // Error: p was moved
    print(h.x)                          // OK: access through h
}
```

This is useful when you create a value and later decide it needs to live on the heap (e.g., to store in a data structure).

### Deep Copy with `.copy()`

To create an independent copy (both remain valid), use `.copy()`. This is a built-in method on `Own<T>`.

```slang
main = () {
    val p = Heap.new(Point{ 1, 2 })
    val q = p.copy()                    // deep copy, new allocation

    print(p.x)                          // OK: p still valid
    print(q.x)                          // OK: q is independent
}                                       // both freed independently
```

**Safe navigation with `.copy()`:** For nullable pointers, `?.copy()` returns `Own<T>?`:
```slang
main = () {
    val p: Own<Point>? = maybeGetPoint()
    val q: Own<Point>? = p?.copy()      // q is null if p is null, otherwise deep copy
}
```

**`.copy()` is only for `Own<T>`:** Stack-allocated copyable types use assignment to copy. Using `.copy()` on a stack value is an error:
```slang
main = () {
    val p = Point{ 1, 2 }              // Stack-allocated, copyable
    val q = p                          // Copy via assignment - both valid
    // val r = p.copy()                // Error: .copy() is only for Own<T>

    val h = Heap.new(Point{ 1, 2 })    // Heap-allocated
    val i = h.copy()                   // OK: deep copy of owned pointer
}
```

**Nested structures:** `.copy()` performs a deep copy, recursively copying all `Own<T>` fields.

```slang
Container = struct {
    val data: Own<Point>
}

main = () {
    val c = Heap.new(Container{ Heap.new(Point{ 1, 2 }) })
    val d = c.copy()                    // deep copy

    // c and d are completely independent
    // c.data and d.data point to different Point allocations
    print(c.data.x)                     // prints: 1
    print(d.data.x)                     // prints: 1

    // Modifying one doesn't affect the other
    // (if fields were var)
}                                       // c and d freed independently, including their nested data
```

### Auto-Free on Reassignment

When reassigning a `var` pointer, the old value is automatically freed.

```slang
main = () {
    var p = Heap.new(Point{ 1, 2 })    // p owns Point{1,2}
    p = Heap.new(Point{ 3, 4 })        // Point{1,2} auto-freed
                                       // p now owns Point{3,4}
}                                      // Point{3,4} freed here
```

## Returning Pointers

Functions can return `Own<T>` - ownership transfers to the caller.

```slang
createPoint = (x: i64, y: i64) -> Own<Point> {
    Heap.new(Point{ x, y })             // ownership transferred to caller
}

main = () {
    val p = createPoint(10, 20)         // main now owns p
    print(p.x)
}                                       // p freed here
```

## Pointers in Structs

Struct fields with `Own<T>` types are **owned** by the struct.

```slang
Container = struct {
    val data: Own<Point>
}

main = () {
    val point = Heap.new(Point{ 1, 2 })
    val container = Container{ point }  // point moves into container

    // print(point.x)                   // Error: point was moved
    print(container.data.x)             // OK: access through container
}                                       // container freed, data freed with it
```

## Nullable Pointers

Nullable pointers follow the same ownership rules.

```slang
Node = struct {
    val value: i64
    var next: Own<Node>?                // nullable, owned by this node
}

main = () {
    val n2 = Heap.new(Node{ 20, null })
    val n1 = Heap.new(Node{ 10, n2 })   // n2 moves into n1.next

    print(n1.value)                     // 10
    print(n1.next?.value)               // 20
}                                       // n1 freed, recursively frees n1.next
```

## Summary Table

| Operation | Syntax | Effect |
|-----------|--------|--------|
| Allocate literal | `Heap.new(Point{ 1, 2 })` | Returns `Own<T>`, caller owns it |
| Stack to heap | `Heap.new(p)` | Moves stack value to heap, returns `Own<T>` |
| Assign | `val q = p` | Moves ownership, `p` invalid |
| Copy | `p.copy()` | Deep copy, both valid |
| Pass to `Ref<T>` param | `f(p)` | Borrow (immutable) |
| Pass to `var Ref<T>` param | `f(p)` | Borrow (mutable) |
| Pass to `Own<T>` param | `f(p)` | Transfers ownership |
| Return `Own<T>` | `return p` | Transfers to caller |
| Reassign `var` | `p = new` | Old value auto-freed |
| Scope exit | `}` | Owner's memory freed |

## Compiler Errors

```slang
// Error: use after move
val p = Heap.new(Point{ 1, 2 })
val q = p
print(p.x)                              // Error: 'p' was moved to 'q'

// Error: cannot mutate through immutable Ref
readPoint = (p: Ref<Point>) {
    p.x = 100                           // Error: p is not var
}

// Error: cannot store Ref<T> in struct
BadStruct = struct {
    val cached: Ref<Point>              // Error: Ref<T> cannot be stored
}

// Error: cannot return Ref<T>
bad = (p: Ref<Point>) -> Ref<Point> {
    p                                   // Error: cannot return Ref<T>
}

// Error: cannot mutate through val binding
main = () {
    val p = Heap.new(Point{ 1, 2 })
    p.x = 10                            // Error: p is val, cannot mutate
}
```

# Edge Cases & Rules

This section documents specific rules to prevent undefined behavior.

## Rule: No Self-Referential Structures

Cannot assign a pointer into a field of the same struct instance.

```slang
Node = struct {
    var next: Own<Node>?
}

main = () {
    var n = Heap.new(Node{ null })
    n.next = n                         // Error: cannot create self-reference
}
```

**Rationale:** Would cause infinite loop or double-free during deallocation.

## Rule: `Ref<T>` Cannot Be Stored or Returned

`Ref<T>` can only appear as function parameter types. Cannot be stored in variables, returned, or used in struct fields.

```slang
// ✅ OK: Ref<T> as parameter
printPoint = (p: Ref<Point>) { print(p.x) }

// ❌ Error: Ref<T> as return type
bad1 = (p: Ref<Point>) -> Ref<Point> { p }

// ❌ Error: Ref<T> as local variable type
main = () {
    val p = Heap.new(Point{ 1, 2 })
    val borrowed: Ref<Point> = p       // Error: cannot store Ref<T>
}

// ❌ Error: Ref<T> as struct field
Cache = struct {
    val ref: Ref<Point>                // Error: Ref<T> cannot be stored
}
```

**Rationale:** Prevents dangling pointers from outliving their source.

## Rule: Conditional Move Invalidates

If a variable is moved in any branch, it's invalid after the entire if/else. This applies to all conditional constructs including short-circuit operators and conditional expressions.

```slang
main = () {
    val p = Heap.new(Point{ 1, 2 })

    if (condition) {
        val q = p                      // moves p
    }

    print(p.x)                         // Error: p may have been moved
}
```

**Short-circuit operators:**
```slang
main = () {
    val p = Heap.new(Point{ 1, 2 })

    val result = condition || consume(p)  // p conditionally moved
    print(p.x)                         // Error: p may have been moved

    val q = Heap.new(Point{ 3, 4 })
    val result2 = condition && consume(q) // q conditionally moved
    print(q.x)                         // Error: q may have been moved
}
```

**Conditional expressions:**
```slang
main = () {
    val p = Heap.new(Point{ 1, 2 })
    val q = Heap.new(Point{ 3, 4 })

    val r = condition ? p : q          // Both p and q conditionally moved
    print(p.x)                         // Error: p may have been moved
    print(q.x)                         // Error: q may have been moved
    print(r.x)                         // OK: r owns whichever was selected
}
```

**Rationale:** Conservative rule avoids complex per-branch tracking.

## Rule: No Move in Loop

Moving a pointer inside a loop body is an error.

```slang
main = () {
    val p = Heap.new(Point{ 1, 2 })

    for (var i = 0; i < 3; i = i + 1) {
        val q = p                      // Error: cannot move in loop
    }
}
```

**Rationale:** Second iteration would use already-moved variable.

## Rule: No Partial Moves

Cannot move individual fields out of a struct.

```slang
Outer = struct {
    val inner: Own<Inner>
}

main = () {
    val outer = Outer{ Heap.new(Inner{ 42 }) }
    val extracted = outer.inner        // Error: cannot move field
}
```

**Rationale:** Keeps structs fully valid or fully invalid.

## Rule: Mutable Borrow Requires `var` Source

To create a mutable borrow (`var Ref<T>`), the source must be a `var` binding.

```slang
mutate = (var p: Ref<Point>) {
    p.x = 10
}

main = () {
    val p = Heap.new(Point{ 1, 2 })   // val binding
    mutate(p)                          // Error: cannot create mutable borrow from val binding

    var q = Heap.new(Point{ 1, 2 })   // var binding
    mutate(q)                          // OK: q is var
}
```

**Rationale:** A `val` binding promises immutability. Allowing mutable borrows would violate that promise.

## Rule: Borrow Exclusivity

A value can have **either** one mutable borrow (`var Ref<T>`) **or** any number of immutable borrows (`Ref<T>`), but not both simultaneously.

```slang
// ✅ OK: multiple immutable borrows
readBoth = (a: Ref<Point>, b: Ref<Point>) {
    print(a.x + b.x)
}
main = () {
    val p = Heap.new(Point{ 1, 2 })
    readBoth(p, p)                     // OK: both are immutable borrows
}

// ❌ Error: multiple mutable borrows
bothMutate = (var a: Ref<Point>, var b: Ref<Point>) {
    a.x = 10
    b.x = 20
}
main = () {
    var p = Heap.new(Point{ 1, 2 })
    bothMutate(p, p)                   // Error: cannot have two mutable borrows
}

// ❌ Error: mutable + immutable borrow
mixedBorrow = (var a: Ref<Point>, b: Ref<Point>) {
    a.x = b.x + 1
}
main = () {
    var p = Heap.new(Point{ 1, 2 })
    mixedBorrow(p, p)                  // Error: cannot mix mutable and immutable borrows
}
```

**Rationale:** Prevents data races and confusing mutation conflicts.

## Rule: Nullable Pointers Move Too

Moving a nullable pointer moves it regardless of runtime null state.

```slang
main = () {
    val p: Own<Point>? = maybeGetPoint()

    if (p != null) {
        val q = p                      // moves p
    }

    print(p?.x)                        // Error: p was moved
}
```

**Rationale:** Compile-time tracking doesn't depend on runtime values.

## Rule: Nullable Pointer Smart Cast

Inside a null check, nullable pointers are smart cast to their non-null type. In the `else` branch, the pointer is known to be null.

```slang
main = () {
    val p: Own<Point>? = maybeGetPoint()

    if (p != null) {
        // p is smart cast to Own<Point> inside this block
        print(p.x)                     // OK: no ?. needed
        print(p.y)                     // OK: direct access

        // Normal ownership rules still apply
        consume(p)                     // moves p
        // print(p.x)                  // Error: p was moved
    } else {
        // p is known to be null here
        print("no point available")
    }

    // After the block, conditional move rule applies
    // print(p?.x)                     // Error: p may have been moved
}
```

**Reassignment preserves validity:**
```slang
main = () {
    var p: Own<Point>? = maybeGetPoint()

    if (p != null) {
        p = null                       // reassign, not move
    }

    print(p?.x)                        // OK: p wasn't moved, just reassigned
}
```

**While loops:** Smart casting also works in `while` loop bodies:
```slang
main = () {
    var p: Own<Point>? = maybeGetPoint()

    while (p != null) {
        print(p.x)                     // p: Own<Point> (smart cast)
        p = getNextPoint()             // may reassign to null
    }
}
```

**Nested null checks:** When accessing nullable fields on smart-casted values:
```slang
Container = struct {
    val data: Own<Point>?
}

main = () {
    val c: Own<Container>? = maybeGetContainer()

    if (c != null) {
        // c: Own<Container> (smart cast)
        if (c.data != null) {
            // c.data: Ref<Point> (auto-borrow + smart cast)
            print(c.data.x)            // OK: direct access
        }
    }
}
```

**Rationale:** Smart casting eliminates verbose `?.` access when null has been ruled out.

## Rule: Mutability Requires `var` Binding

To mutate through a pointer, both the binding must be `var` and the field must be `var`.

```slang
Point = struct {
    val x: i64      // immutable field
    var y: i64      // mutable field
}

main = () {
    val p = Heap.new(Point{ 1, 2 })
    // p.x = 10                        // Error: x is val field
    // p.y = 20                        // Error: p is val binding

    var q = Heap.new(Point{ 1, 2 })
    // q.x = 10                        // Error: x is val field
    q.y = 20                           // OK: q is var and y is var
}
```

**Rationale:** Consistent with struct mutability rules; `val`/`var` applies uniformly.

## Rule: Closures Cannot Capture `Ref<T>`

Closures can be stored or returned, so they cannot capture `Ref<T>` (would violate "no storing Ref").

```slang
// ❌ Error: cannot capture Ref<T>
useRef = (p: Ref<Point>) {
    val f = () {
        print(p.x)                     // Error: cannot capture Ref<T>
    }
}

// ✅ OK: capture Own<T> (moves ownership into closure)
main = () {
    val p = Heap.new(Point{ 1, 2 })

    val f = () {
        print(p.x)                     // p moved into closure
    }

    // print(p.x)                      // Error: p was moved into closure
    f()                                // OK: closure owns p
}

// ✅ OK: pass Ref<T> as parameter instead of capturing
forEach = (arr: Ref<Array<i64>>, f: (i64) -> void) {
    for (var i = 0; i < len(arr); i = i + 1) {
        f(arr[i])
    }
}
```

**Rationale:** Closures can escape their creating scope; captured `Ref<T>` could dangle.

## Rule: Generics Cannot Store `Ref<T>`

Type parameters can only be instantiated with `Ref<T>` if used exclusively in function parameter positions.

```slang
// ✅ OK: Own<T> as type argument for fields
List = struct<T> {
    var items: Array<T>
}

main = () {
    var list: List<Own<Point>> = List{ [] }
    list.items = append(list.items, Heap.new(Point{ 1, 2 }))
}

// ❌ Error: Ref<T> as type argument for fields
Cache = struct<T> {
    val item: T                        // if T = Ref<Point>, this is invalid
}

main = () {
    val p = Heap.new(Point{ 1, 2 })
    val c: Cache<Ref<Point>> = ...     // Error: Ref<T> cannot be stored
}
```

**Rationale:** Consistent with "Ref<T> cannot be stored" rule.

## Rule: Temporary Lifetimes

Temporaries (values returned from function calls) live until the end of the statement.

```slang
createPoint = () -> Own<Point> {
    Heap.new(Point{ 10, 20 })
}

main = () {
    // Temporary Own<Point> lives for the full statement
    val x = createPoint().x            // OK: temp lives until semicolon
    print(x)                           // prints: 10
    // Temporary freed after previous statement

    // Chained access works
    print(createPoint().x + createPoint().y)  // OK: both temps live until statement end
    // Both temps freed here

    // Nested temporaries
    val v = getOuter().inner.field     // OK: all temps live until statement end
}
```

**Rationale:** Standard temporary lifetime semantics; matches C++, Rust, Swift.

## Rule: No Self-Assignment

Assigning an `Own<T>` variable to itself is a compile-time error.

```slang
main = () {
    var p = Heap.new(Point{ 1, 2 })
    p = p                              // Error: cannot assign variable to itself
}
```

**Rationale:** Self-assignment would move the value out (invalidating `p`) before the assignment drops the old value (which was already moved), causing use-after-free.

## Rule: No Overlapping Moves in Assignment

The left-hand side and right-hand side of an assignment cannot share paths when `Own<T>` moves are involved.

```slang
Container = struct {
    var data: Own<Data>
}

main = () {
    var c = Heap.new(Container{ Heap.new(Data{}) })

    c = c.data                         // Error: 'c' appears in both sides of move
    c.data = c                         // Error: 'c' appears in both sides of move
}
```

**Rationale:** Moving from a path that overlaps with the assignment target causes use-after-free or double-free.

## Rule: Left-to-Right Evaluation Order

Expressions are evaluated left-to-right. If a variable is moved, any subsequent use in the same expression is an error.

```slang
main = () {
    val p = Heap.new(Point{ 1, 2 })

    // Assuming consume takes Own<T> (moves) and read takes Ref<T> (borrows)
    consume(p, p)                      // Error: 'p' moved in first argument, used in second
    consume(p, read(p))                // Error: 'p' moved in first argument, read(p) uses moved value
    consume(read(p), p)                // Error: read(p) evaluates first, but 'p' moved in second arg
}
```

**Note:** Multiple immutable borrows in one expression are fine:
```slang
main = () {
    val p = Heap.new(Point{ 1, 2 })
    readBoth(p, p)                     // OK: both are immutable borrows
}
```

**Rationale:** Consistent evaluation order makes move tracking predictable and prevents accidental double-moves.

## Rule: No Mixed Borrow and Move

Cannot borrow and move the same value in a single function call.

```slang
mixed = (r: Ref<Point>, p: Own<Point>) {
    print(r.x)
    // p is freed at end of function
}

main = () {
    val p = Heap.new(Point{ 1, 2 })
    mixed(p, p)                        // Error: cannot borrow and move same value
    // First argument borrows p, second moves it
    // The borrow would dangle when p is moved
}
```

**Rationale:** A borrow creates a reference; moving invalidates what the reference points to.

## Rule: Early Return Cleanup

When a function returns early, all live `Own<T>` values are dropped in reverse declaration order.

```slang
main = () {
    val a = Heap.new(Point{ 1, 2 })   // a is live
    if (condition) {
        return                         // Drop: a
    }
    val b = Heap.new(Point{ 3, 4 })   // a, b are live
    if (condition2) {
        consume(a)                     // a is moved, no longer live
        return                         // Drop: b (not a, it was moved)
    }
}                                      // Drop: b, a (reverse order)
```

**Rationale:** LIFO cleanup order matches stack semantics and ensures proper resource cleanup regardless of control flow.

## Rule: Loop Iteration Borrows

Iterating over a container borrows it; elements are borrowed or copied based on type.

```slang
main = () {
    val arr = Heap.new([1, 2, 3, 4, 5])

    // Borrows arr; x is a copy (i64 is copyable)
    for x in arr {
        print(x)
    }

    print(arr[0])                      // OK: arr still valid
}

// For pointer elements: x borrows each element
main = () {
    val points: Own<Array<Own<Point>>> = Heap.new([
        Heap.new(Point{ 1, 2 }),
        Heap.new(Point{ 3, 4 })
    ])

    // x borrows each element (cannot move out of array)
    for x in points {
        print(x.x)                     // x: Ref<Point> (borrowed)
    }

    // Array still owns all points
    print(points[0].x)                 // OK
}

// Mutable iteration: var array produces var Ref
main = () {
    var points = Heap.new([
        Heap.new(Point{ 1, 2 }),
        Heap.new(Point{ 3, 4 })
    ])

    // x is var Ref<Point> since points is var
    for x in points {
        x.x = x.x * 2                  // OK: can mutate through var Ref
    }

    print(points[0].x)                 // prints: 2
}
```

**Rationale:** Moving elements out during iteration would leave holes. Borrowing is safe. Mutability inherits from the source binding.

## Rule: Cannot Reassign While Borrowed

Cannot reassign an owned pointer while any borrows of it exist. This includes during iteration.

```slang
main = () {
    var arr = Heap.new([1, 2, 3])

    for x in arr {
        arr = Heap.new([4, 5, 6])      // Error: cannot reassign 'arr' while borrowed
        // The for loop borrows arr for the duration of iteration
    }

    // After the loop, arr can be reassigned
    arr = Heap.new([7, 8, 9])          // OK: no active borrows
}
```

**Rationale:** Reassignment drops the old value. Dropping a borrowed value would invalidate the borrow.

## Rule: Operators Auto-Dereference `Own<primitive>`

Arithmetic and comparison operators auto-dereference `Own<T>` for primitive types. The result is the primitive type, not a pointer.

```slang
main = () {
    val p = Heap.new(42)
    val sum = p + 1                    // OK: auto-derefs, sum is i64 (43)
    val product = p * 2                // OK: auto-derefs, product is i64 (84)
    val isLarge = p > 100              // OK: auto-derefs, isLarge is bool (false)

    // The pointer itself is unchanged
    print(p + p)                       // OK: prints 84 (42 + 42)

    // Comparison between pointers is still identity comparison
    val q = Heap.new(42)
    print(p == q)                      // false: different allocations (identity)
    print(p == 42)                     // true: auto-derefs p, compares values
}
```

**Note:** Assignment operators do not auto-dereference. To modify the value, use a struct wrapper:
```slang
Counter = struct {
    var value: i64
}

main = () {
    var c = Heap.new(Counter{ 42 })
    c.value = c.value + 1              // OK: modify through field access
    print(c.value)                     // prints: 43
}
```

**Rationale:** Auto-dereferencing for operators provides convenience for the common case of arithmetic on heap-allocated primitives.

## Rule: `null` as Function Argument

`null` can be passed directly to functions expecting nullable pointer parameters.

```slang
foo = (p: Own<Point>?) { ... }
bar = (r: Ref<Point>?) { ... }

main = () {
    foo(null)                          // OK: null is valid for Own<T>?
    bar(null)                          // OK: null is valid for Ref<T>?
}
```

**Rationale:** Nullable parameters should accept null values directly.

## Rule: `break` and `continue` Cleanup

`break` and `continue` statements drop all live owned values in the current scope before exiting.

```slang
main = () {
    for i in range(10) {
        val p = Heap.new(Point{ i, i })
        if (condition) {
            break                      // Drops p before exiting loop
        }
        val q = Heap.new(Point{ i, i + 1 })
        if (otherCondition) {
            continue                   // Drops q and p before next iteration
        }
    }                                  // Normal exit also drops loop-local values
}
```

**Rationale:** All control flow paths must properly clean up owned resources.

## Rule: Array of Nullable Pointers

For `Array<Own<T>?>`, indexing returns `Ref<T>?` (nullable borrow).

```slang
main = () {
    val arr: Own<Array<Own<Point>?>> = Heap.new([
        Heap.new(Point{ 1, 2 }),
        null,
        Heap.new(Point{ 3, 4 })
    ])

    val p = arr[0]                     // p: Ref<Point>? (non-null borrow)
    val q = arr[1]                     // q: Ref<Point>? (null)

    print(p?.x)                        // prints: 1
    print(q?.x)                        // prints: null
}
```

**Rationale:** The borrow inherits the nullability of the element type.

## Rule: Array Literal Type Inference

Array literals containing both `Own<T>` and `null` infer element type `Own<T>?`.

```slang
main = () {
    // Inferred as Array<Own<Point>?>
    val arr = [
        Heap.new(Point{ 1, 2 }),
        null,
        Heap.new(Point{ 3, 4 })
    ]

    // Explicit type annotation works too
    val arr2: Array<Own<Point>?> = [Heap.new(Point{ 5, 6 }), null]
}
```

**Rationale:** Consistent with nullable type inference elsewhere in the language.

## Rule: Implicit Return Moves

Implicit return (expression as last statement) moves ownership just like explicit `return`.

```slang
// These are equivalent:
createExplicit = () -> Own<Point> {
    val p = Heap.new(Point{ 1, 2 })
    return p                           // Explicit return, moves p
}

createImplicit = () -> Own<Point> {
    val p = Heap.new(Point{ 1, 2 })
    p                                  // Implicit return, moves p
}

// Direct allocation works too
createDirect = () -> Own<Point> {
    Heap.new(Point{ 1, 2 })            // Implicit return of temporary
}
```

**Rationale:** Implicit and explicit returns should have identical ownership semantics.

## Rule: Pass-Through Ownership

A function can take ownership and immediately return it (pass-through).

```slang
// Valid: takes ownership, returns same value
identity = (p: Own<Point>) -> Own<Point> {
    p                                  // Ownership transfers through
}

// Useful for conditional wrapping
maybeWrap = (p: Own<Point>, shouldWrap: bool) -> Own<Container> {
    if (shouldWrap) {
        Heap.new(Container{ p })       // p moves into Container
    } else {
        Heap.new(Container{ p })       // same
    }
}

main = () {
    val p = Heap.new(Point{ 1, 2 })
    val q = identity(p)                // p moves to identity, then to q
    // print(p.x)                      // Error: p was moved
    print(q.x)                         // OK: q owns it now
}
```

**Rationale:** Ownership is tracked through function boundaries; pass-through is a valid pattern.

## Understanding `var` in Different Contexts

The `var` keyword means "mutable" in two contexts, which can be confusing:

**1. Variable declarations:** `var` means the binding can be reassigned
```slang
var x = 5           // x can be reassigned
x = 10              // OK

val y = 5           // y cannot be reassigned
// y = 10           // Error
```

**2. Reference parameters:** `var` means you can mutate through the reference
```slang
// var before parameter: can mutate the referenced data
mutate = (var p: Ref<Point>) {
    p.x = 10        // OK: can modify through var Ref
}

// Without var: read-only access
read = (p: Ref<Point>) {
    print(p.x)      // OK: can read
    // p.x = 10     // Error: p is not var
}
```

**Common confusion:**
```slang
// This does NOT mean p can be reassigned inside the function
// It means you can mutate the data p points to
mutate = (var p: Ref<Point>) {
    p.x = 10        // Mutates the Point that p references
    // p = ...      // This would be reassigning the parameter itself (different concept)
}
```

**The key insight:** In both cases, `var` grants write permission:
- For bindings: permission to write a new value to the variable
- For references: permission to write through the reference to the underlying data

# Patterns for Self-Referential Structures

Single ownership prevents true cycles. This section documents patterns for modeling structures that would traditionally use cycles or back-references.

## The Limitation

These common patterns don't work with single ownership:

```slang
// ❌ Doubly-linked list - two owners for each node
Node = struct {
    var next: Own<Node>?
    var prev: Own<Node>?       // Error: would need two owners
}

// ❌ Tree with parent pointer
TreeNode = struct {
    var children: Array<Own<TreeNode>>
    var parent: Own<TreeNode>? // Error: parent owns child, child can't own parent
}

// ❌ Graph with cycles
GraphNode = struct {
    var neighbors: Array<Own<GraphNode>>   // Error: cycles = multiple owners
}
```

## Pattern 1: Index-Based References (Arena Pattern)

Store all nodes in an array. Use indices instead of pointers for back-references.

```slang
// Doubly-linked list using indices
DLLNode = struct {
    val value: i64
    var next: i64              // index into nodes array (-1 = none)
    var prev: i64              // index into nodes array (-1 = none)
}

DoublyLinkedList = struct {
    var nodes: Own<Array<DLLNode>>
    var head: i64
    var tail: i64
}

// Helper to get node by index
getNode = (list: Ref<DoublyLinkedList>, idx: i64) -> Ref<DLLNode> {
    list.nodes[idx]
}

main = () {
    var list = Heap.new(DoublyLinkedList{
        Heap.new([
            DLLNode{ 10, 1, -1 },     // [0]: value=10, next=1, prev=none
            DLLNode{ 20, 2, 0 },      // [1]: value=20, next=2, prev=0
            DLLNode{ 30, -1, 1 }      // [2]: value=30, next=none, prev=1
        ]),
        0,   // head index
        2    // tail index
    })

    // Forward traversal
    var idx = list.head
    while (idx != -1) {
        print(list.nodes[idx].value)
        idx = list.nodes[idx].next
    }
    // prints: 10, 20, 30

    // Backward traversal
    idx = list.tail
    while (idx != -1) {
        print(list.nodes[idx].value)
        idx = list.nodes[idx].prev
    }
    // prints: 30, 20, 10
}
```

**Advantages:**
- No ownership complexity
- Cache-friendly (contiguous memory)
- Easy to serialize

**Disadvantages:**
- Manual index management
- Can't free individual nodes (need compaction)
- Index invalidation if array is modified

## Pattern 2: Parent via Parameter (Context Pattern)

Pass parent/context as a parameter during traversal rather than storing it.

```slang
TreeNode = struct {
    val value: i64
    var children: Array<Own<TreeNode>>
    // No parent field - passed during traversal
}

// Parent available as parameter, not stored
traverseWithParent = (
    node: Ref<TreeNode>,
    parent: Ref<TreeNode>?,
    visit: (Ref<TreeNode>, Ref<TreeNode>?) -> void
) {
    visit(node, parent)

    for (var i = 0; i < len(node.children); i = i + 1) {
        traverseWithParent(node.children[i], node, visit)
    }
}

// Find path to root by walking up via recursion
findDepth = (node: Ref<TreeNode>, parent: Ref<TreeNode>?) -> i64 {
    if (parent == null) {
        0
    } else {
        // Parent is available in this scope
        1  // Would need different pattern to actually walk up
    }
}

main = () {
    var tree = Heap.new(TreeNode{ 1, [
        Heap.new(TreeNode{ 2, [] }),
        Heap.new(TreeNode{ 3, [
            Heap.new(TreeNode{ 4, [] })
        ]})
    ]})

    traverseWithParent(tree, null, (node, parent) {
        if (parent != null) {
            print(parent.value)
            print(" -> ")
        }
        print(node.value)
    })
}
```

**Advantages:**
- Clean ownership model
- No cycles possible
- Natural for recursive algorithms

**Disadvantages:**
- Can't navigate upward without traversal context
- Must thread parent through all operations

## Pattern 3: Edge List (Graph Pattern)

For graphs, separate nodes from edges. Store edges as (from, to) pairs.

```slang
Graph = struct {
    var nodes: Array<NodeData>
    var edges: Array<Edge>
}

NodeData = struct {
    val id: i64
    val label: string
}

Edge = struct {
    val from: i64              // index into nodes
    val to: i64                // index into nodes
    val weight: i64
}

// Get all neighbors of a node
neighbors = (g: Ref<Graph>, nodeIdx: i64) -> Array<i64> {
    var result: Array<i64> = []
    for (var i = 0; i < len(g.edges); i = i + 1) {
        if (g.edges[i].from == nodeIdx) {
            result = append(result, g.edges[i].to)
        }
    }
    result
}

// BFS traversal
bfs = (g: Ref<Graph>, start: i64) {
    var visited: Array<bool> = [false; len(g.nodes)]
    var queue: Array<i64> = [start]

    while (len(queue) > 0) {
        val current = queue[0]
        queue = rest(queue)  // remove first element

        if (!visited[current]) {
            visited[current] = true
            print(g.nodes[current].label)

            val neighs = neighbors(g, current)
            for (var i = 0; i < len(neighs); i = i + 1) {
                queue = append(queue, neighs[i])
            }
        }
    }
}

main = () {
    val graph = Graph{
        [
            NodeData{ 0, "A" },
            NodeData{ 1, "B" },
            NodeData{ 2, "C" }
        ],
        [
            Edge{ 0, 1, 1 },   // A -> B
            Edge{ 1, 2, 1 },   // B -> C
            Edge{ 2, 0, 1 }    // C -> A (cycle via indices!)
        ]
    }

    bfs(graph, 0)  // prints: A, B, C
}
```

**Advantages:**
- Handles arbitrary cycles
- Easy to add/remove edges
- Standard graph representation

**Disadvantages:**
- O(E) to find neighbors (can optimize with adjacency list)
- Less intuitive than pointer-based graphs

## Pattern 4: Build-As-You-Go Graph

For graphs built incrementally, use mutable arrays with index-based references.

```slang
Graph = struct {
    var nodes: Array<NodeData>         // growable
    var edges: Array<Edge>             // growable
    var freeList: Array<i64>           // indices of deleted nodes for reuse
}

NodeData = struct {
    val id: i64
    val label: string
    val weight: i64
    var deleted: bool                  // tombstone flag
}

Edge = struct {
    val from: i64                      // index into nodes
    val to: i64                        // index into nodes
    var deleted: bool                  // tombstone flag
}

// Constructor
newGraph = () -> Own<Graph> {
    Heap.new(Graph{ [], [], [] })
}

// Add node, returns its index (reuses deleted slots)
addNode = (var g: Ref<Graph>, label: string, weight: i64) -> i64 {
    if (len(g.freeList) > 0) {
        // Reuse a deleted slot
        val id = g.freeList[len(g.freeList) - 1]
        g.freeList = dropLast(g.freeList)
        g.nodes[id] = NodeData{ id, label, weight, false }
        id
    } else {
        // Append new node
        val id = len(g.nodes)
        g.nodes = append(g.nodes, NodeData{ id, label, weight, false })
        id
    }
}

// Add directed edge
addEdge = (var g: Ref<Graph>, from: i64, to: i64) {
    g.edges = append(g.edges, Edge{ from, to, false })
}

// Add undirected edge (two directed edges)
connect = (var g: Ref<Graph>, a: i64, b: i64) {
    addEdge(g, a, b)
    addEdge(g, b, a)
}

// Delete a node (marks as deleted, adds to free list)
deleteNode = (var g: Ref<Graph>, nodeIdx: i64) {
    if (nodeIdx >= 0 && nodeIdx < len(g.nodes) && !g.nodes[nodeIdx].deleted) {
        // Mark node as deleted
        g.nodes[nodeIdx].deleted = true

        // Add to free list for reuse
        g.freeList = append(g.freeList, nodeIdx)

        // Mark all edges involving this node as deleted
        for (var i = 0; i < len(g.edges); i = i + 1) {
            if (g.edges[i].from == nodeIdx || g.edges[i].to == nodeIdx) {
                g.edges[i].deleted = true
            }
        }
    }
}

// Delete a specific edge
deleteEdge = (var g: Ref<Graph>, from: i64, to: i64) {
    for (var i = 0; i < len(g.edges); i = i + 1) {
        if (g.edges[i].from == from && g.edges[i].to == to && !g.edges[i].deleted) {
            g.edges[i].deleted = true
            return
        }
    }
}

// Check if node is valid (exists and not deleted)
isValidNode = (g: Ref<Graph>, nodeIdx: i64) -> bool {
    nodeIdx >= 0 && nodeIdx < len(g.nodes) && !g.nodes[nodeIdx].deleted
}

// Check if edge exists (and not deleted)
hasEdge = (g: Ref<Graph>, from: i64, to: i64) -> bool {
    for (var i = 0; i < len(g.edges); i = i + 1) {
        if (g.edges[i].from == from && g.edges[i].to == to && !g.edges[i].deleted) {
            return true
        }
    }
    false
}

// Get all outgoing neighbors (skips deleted)
outNeighbors = (g: Ref<Graph>, nodeIdx: i64) -> Array<i64> {
    var result: Array<i64> = []
    for (var i = 0; i < len(g.edges); i = i + 1) {
        if (g.edges[i].from == nodeIdx && !g.edges[i].deleted) {
            val neighbor = g.edges[i].to
            if (isValidNode(g, neighbor)) {
                result = append(result, neighbor)
            }
        }
    }
    result
}

// Count active (non-deleted) nodes
nodeCount = (g: Ref<Graph>) -> i64 {
    var count = 0
    for (var i = 0; i < len(g.nodes); i = i + 1) {
        if (!g.nodes[i].deleted) {
            count = count + 1
        }
    }
    count
}

// Example: Build a social network with deletion
main = () {
    var network = newGraph()

    // Add people
    val alice = addNode(network, "Alice", 25)   // index 0
    val bob = addNode(network, "Bob", 30)       // index 1
    val carol = addNode(network, "Carol", 28)   // index 2

    // Add friendships (bidirectional)
    connect(network, alice, bob)
    connect(network, bob, carol)

    print(nodeCount(network))              // 3

    // Bob leaves the network
    deleteNode(network, bob)

    print(nodeCount(network))              // 2
    print(hasEdge(network, alice, bob))    // false (edge was deleted)
    print(isValidNode(network, bob))       // false

    // Alice's friends after Bob left
    val aliceFriends = outNeighbors(network, alice)
    print(len(aliceFriends))               // 0 (Bob was her only friend)

    // Add new person - reuses Bob's slot (index 1)
    val dave = addNode(network, "Dave", 35)
    print(dave)                            // 1 (reused index)
    print(nodeCount(network))              // 3

    // Connect Dave
    connect(network, alice, dave)
    connect(network, carol, dave)

    // Alice's friends now
    val newFriends = outNeighbors(network, alice)
    for (var i = 0; i < len(newFriends); i = i + 1) {
        print(network.nodes[newFriends[i]].label)
    }
    // prints: Dave

    // Can also delete just an edge
    deleteEdge(network, carol, dave)
    print(hasEdge(network, carol, dave))   // false
    print(hasEdge(network, dave, carol))   // true (only one direction deleted)
}
```

**Advantages:**
- Intuitive API for building graphs
- Handles cycles naturally via indices
- Nodes and edges can be added at any time
- Deletion via tombstones preserves index stability
- Free list enables slot reuse (reduces memory fragmentation)

**Disadvantages:**
- Tombstones consume memory until compaction
- O(E) edge lookups (can optimize with adjacency list)
- All operations must check `deleted` flag

## Pattern 5: Separate Ownership from Navigation

Use owned pointers for the primary structure, indices for secondary references.

```slang
// Tree with efficient parent lookup via separate index
TreeNode = struct {
    val id: i64
    val value: i64
    var children: Array<Own<TreeNode>>
}

Tree = struct {
    var root: Own<TreeNode>
    var parentMap: Own<Map<i64, i64>>      // child_id -> parent_id
}

buildTree = () -> Own<Tree> {
    val child1 = Heap.new(TreeNode{ 1, 10, [] })
    val child2 = Heap.new(TreeNode{ 2, 20, [] })
    val root = Heap.new(TreeNode{ 0, 0, [child1, child2] })

    var parents = Heap.new(Map{})
    parents.set(1, 0)  // child1's parent is root
    parents.set(2, 0)  // child2's parent is root

    Heap.new(Tree{ root, parents })
}

getParentId = (tree: Ref<Tree>, nodeId: i64) -> i64? {
    tree.parentMap.get(nodeId)
}
```

# Method Receivers (SEP 7 Interaction)

When classes (SEP 7) are implemented, method receivers use explicit ownership types to indicate borrowing vs ownership transfer.

## Receiver Types

```slang
Point = class {
    var x: i64
    var y: i64

    // Immutable borrow - cannot modify self
    magnitude = (self: Ref<Point>) -> i64 {
        sqrt(self.x * self.x + self.y * self.y)
    }

    // Mutable borrow - can modify self, caller keeps ownership
    scale = (var self: Ref<Point>, factor: i64) {
        self.x = self.x * factor
        self.y = self.y * factor
    }

    // Takes ownership - self is consumed
    intoArray = (self: Own<Point>) -> Array<i64> {
        [self.x, self.y]
    }   // self freed here
}

main = () {
    var p = Heap.new(Point{ 3, 4 })

    print(p.magnitude())              // borrows p (immutable)
    p.scale(2)                        // borrows p (mutable)
    print(p.x)                        // prints: 6

    val arr = p.intoArray()           // p moved, consumed
    // print(p.x)                     // Error: p was moved
}
```

## Static Factory Methods

Static methods (no `self` parameter) can return `Own<Self>`:

```slang
Point = class {
    var x: i64
    var y: i64

    // Static factory - no self parameter
    static new = (x: i64, y: i64) -> Own<Point> {
        Heap.new(Point{ x, y })
    }

    // Static factory with default
    static origin = () -> Own<Point> {
        Point.new(0, 0)
    }
}

main = () {
    val p = Point.new(10, 20)         // returns Own<Point>
    val origin = Point.origin()
}
```

## Summary

| Receiver Type | Effect | Caller Ownership |
|---------------|--------|------------------|
| `self: Ref<T>` | Immutable borrow | Keeps ownership |
| `var self: Ref<T>` | Mutable borrow | Keeps ownership |
| `self: Own<T>` | Takes ownership | Loses access |

## Method Chaining with Consuming Methods

Method chaining works with consuming methods (`self: Own<T>`). The receiver is moved into the method.

```slang
Point = class {
    var x: i64
    var y: i64

    // Consuming method - self is moved in
    intoArray = (self: Own<Point>) -> Own<Array<i64>> {
        Heap.new([self.x, self.y])
    }   // self freed here
}

main = () {
    // Chaining on temporary - temporary is consumed
    val arr = Heap.new(Point{ 1, 2 }).intoArray()
    print(arr[0])                      // prints: 1

    // Chaining on variable - variable is moved
    val p = Heap.new(Point{ 3, 4 })
    val arr2 = p.intoArray()           // p moved, consumed by method
    // print(p.x)                      // Error: p was moved
    print(arr2[1])                     // prints: 4
}
```

**Rationale:** Temporaries from function calls live until the end of the statement, allowing method chaining. The consuming method takes ownership of the temporary or variable.

## Future Work: Additional Pointer Types

If these patterns prove too limiting, future SEPs may introduce:

### Weak Pointers (`Weak<T>`)

Non-owning references that don't prevent deallocation.

```slang
// Future syntax (not in this SEP)
Node = struct {
    var next: Own<Node>?       // owns next
    var prev: Weak<Node>?      // weak reference, doesn't own
}

main = () {
    var a = Heap.new(Node{ null, null })
    var b = Heap.new(Node{ null, a.weak() })
    a.next = b

    // Later: b.prev.upgrade() returns Own<Node>? (null if freed)
}
```

### Reference Counting (`Rc<T>`)

Shared ownership via reference counting.

```slang
// Future syntax (not in this SEP)
SharedNode = struct {
    val data: i64
    var refs: Array<Rc<SharedNode>>
}

main = () {
    val a = Rc.new(SharedNode{ 1, [] })   // refcount=1
    val b = Rc.new(SharedNode{ 2, [a] })  // a.refcount=2
    // Need weak refs to break cycles
}
```

### Fallible Allocation (`Heap.tryNew`)

For cases where allocation failure should be handled gracefully:

```slang
// Future syntax (not in this SEP)
main = () {
    val maybePoint = Heap.tryNew(Point{ 1, 2 })  // returns Own<Point>?

    if (maybePoint == null) {
        print("allocation failed")
        exit(1)
    }

    print(maybePoint?.x)              // safe access
}
```

### Consuming Iteration (`drain`)

For consuming elements from a collection during iteration:

```slang
// Future syntax (not in this SEP)
main = () {
    var points = Heap.new([
        Heap.new(Point{ 1, 2 }),
        Heap.new(Point{ 3, 4 })
    ])

    // drain() consumes the array, yielding owned elements
    for p in points.drain() {
        consume(p)                    // p: Own<Point>
    }

    // points is now empty/invalid
}
```

These would be separate SEPs building on the ownership foundation established here.

# Implementation

## Built-in Types Infrastructure

This SEP introduces several new built-in types (`Own<T>`, `Ref<T>`, `Heap`) and built-in methods (`.copy()`). This section documents the infrastructure needed to support these and enable easier addition of built-in types in the future.

### Current Built-in Types

Slang currently has the following built-in types:
- **Primitives:** `i64`, `bool`, `string`, `void`
- **Compound:** `Array<T>`, `T?` (nullable)
- **Functions:** Function types like `(i64, i64) -> i64`

These are currently hardcoded in the type system with special-case handling throughout the compiler.

### New Built-in Types in This SEP

| Type | Category | Description |
|------|----------|-------------|
| `Own<T>` | Generic wrapper | Owned pointer type |
| `Ref<T>` | Generic wrapper | Borrowed reference type |
| `Heap` | Singleton type | Built-in allocator with `.new()` method |

### Built-in Type Registry

To enable easier addition of built-in types, introduce a **Built-in Type Registry** in the semantic analyzer:

```go
// compiler/semantic/builtins.go

type BuiltinType struct {
    Name           string
    TypeParams     []string              // e.g., ["T"] for Own<T>
    Methods        map[string]BuiltinMethod
    Constraints    TypeConstraints       // e.g., T must be non-nullable
}

type BuiltinMethod struct {
    Name       string
    Params     []ParamSpec              // Parameter types (may reference type params)
    ReturnType TypeSpec                 // Return type (may reference type params)
    Flags      MethodFlags              // e.g., mutates receiver, consumes receiver
}

type TypeConstraints struct {
    NonNullable bool                    // T cannot be T?
    Copyable    bool                    // T must be copyable
    // Future: other constraints
}

var BuiltinTypes = map[string]BuiltinType{
    "Own": {
        Name:       "Own",
        TypeParams: []string{"T"},
        Methods: map[string]BuiltinMethod{
            "copy": {
                Name:       "copy",
                Params:     []ParamSpec{},
                ReturnType: TypeSpec{Kind: "Own", TypeArg: "T"},
                Flags:      MethodFlags{BorrowsReceiver: true},
            },
        },
        Constraints: TypeConstraints{NonNullable: true},
    },
    "Ref": {
        Name:       "Ref",
        TypeParams: []string{"T"},
        Methods:    map[string]BuiltinMethod{},
        Constraints: TypeConstraints{NonNullable: true},
    },
    "Array": {
        Name:       "Array",
        TypeParams: []string{"T"},
        Methods: map[string]BuiltinMethod{
            "len": {
                Name:       "len",
                Params:     []ParamSpec{},
                ReturnType: TypeSpec{Kind: "i64"},
                Flags:      MethodFlags{BorrowsReceiver: true},
            },
            // Future: push, pop, remove, set, etc.
        },
        Constraints: TypeConstraints{},
    },
}
```

### Built-in Singleton Registry

For types like `Heap` that are singletons with static methods:

```go
// compiler/semantic/builtins.go

type BuiltinSingleton struct {
    Name    string
    Methods map[string]BuiltinStaticMethod
}

type BuiltinStaticMethod struct {
    Name       string
    TypeParams []string                 // Method-level type params
    Params     []ParamSpec
    ReturnType TypeSpec
    Flags      MethodFlags
}

var BuiltinSingletons = map[string]BuiltinSingleton{
    "Heap": {
        Name: "Heap",
        Methods: map[string]BuiltinStaticMethod{
            "new": {
                Name:       "new",
                TypeParams: []string{"T"},              // Inferred from argument
                Params:     []ParamSpec{{Type: "T"}},
                ReturnType: TypeSpec{Kind: "Own", TypeArg: "T"},
                Flags:      MethodFlags{MovesArgument: true},
            },
            // Future: tryNew, alloc, etc.
        },
    },
}
```

### Type Resolution Flow

When the semantic analyzer encounters a type or method call:

1. **Type lookup:** Check `BuiltinTypes` registry first, then user-defined types
2. **Method lookup:** For `expr.method()`, check if `expr`'s type has the method in registry
3. **Singleton lookup:** For `Heap.new()`, check `BuiltinSingletons` registry
4. **Constraint validation:** Ensure type arguments satisfy constraints (e.g., `Own<T?>` fails)

```go
// Example type resolution
func (a *Analyzer) resolveType(typeName string, typeArgs []Type) (Type, error) {
    // Check built-in types first
    if builtin, ok := BuiltinTypes[typeName]; ok {
        if err := validateConstraints(builtin.Constraints, typeArgs); err != nil {
            return nil, err
        }
        return &BuiltinGenericType{
            Name:     typeName,
            TypeArgs: typeArgs,
            Builtin:  builtin,
        }, nil
    }

    // Fall back to user-defined types
    return a.lookupUserType(typeName, typeArgs)
}
```

### Code Generation for Built-ins

Built-in types need special code generation. Extend the codegen registry:

```go
// compiler/codegen/builtins.go

type BuiltinCodegen interface {
    GenerateAllocation(value Expression) string    // For Heap.new
    GenerateDeallocation(ptr Expression) string    // For scope exit
    GenerateCopy(ptr Expression) string            // For .copy()
    GenerateFieldAccess(ptr Expression, field string) string
}

var BuiltinCodegenHandlers = map[string]BuiltinCodegen{
    "Own": &OwnedPointerCodegen{},
    "Ref": &RefPointerCodegen{},
    "Array": &ArrayCodegen{},
}
```

### Benefits of Registry Approach

1. **Centralized definitions:** All built-in types defined in one place
2. **Consistent validation:** Constraints checked uniformly
3. **Easier extension:** Add new built-in types by adding registry entries
4. **Documentation:** Registry serves as authoritative spec for built-ins
5. **Testing:** Can iterate over registry to generate test cases

### Future Built-in Types

This infrastructure enables future additions:

| Type | Description | Methods |
|------|-------------|---------|
| `Weak<T>` | Weak pointer | `.upgrade() -> Own<T>?` |
| `Rc<T>` | Reference counted | `.clone() -> Rc<T>`, `.count() -> i64` |
| `Box<T>` | Simple owned pointer (no custom allocator) | Same as `Own<T>` |
| `Map<K, V>` | Hash map | `.get()`, `.set()`, `.remove()`, `.keys()` |
| `Set<T>` | Hash set | `.add()`, `.remove()`, `.contains()` |
| `Result<T, E>` | Error handling | `.unwrap()`, `.map()`, `.isOk()` |
| `Option<T>` | Explicit optional | `.unwrap()`, `.map()`, `.isSome()` |

### Implementation Priority

1. **Phase 1 (This SEP):** `Own<T>`, `Ref<T>`, `Heap` with manual implementation
2. **Phase 2:** Refactor to use registry approach
3. **Phase 3:** Add more built-in types using registry

The registry approach is recommended but not required for initial implementation. The manual approach can be refactored later.

## Step 1: Lexer Changes

Add token support for pointer syntax:
- Add `Own` and `Ref` keyword tokens
- Add `Heap` keyword token (or treat as built-in identifier)
- Recognize `.new`, `.copy` as method calls

## Step 2: Parser Changes

Extend the parser to handle pointer expressions and types:
- Parse `Own<T>` and `Ref<T>` type syntax
- Parse `Heap.new(expr)` as allocation expression
- Parse `.copy()` method calls
- Parse `var` modifier on function parameters
- Enforce `Ref<T>` only in parameter position

## Step 3: Type System Changes

Add pointer types to the semantic analyzer:
- Add `OwnedPointerType` and `RefPointerType` structs
- `Heap.new(expr)` returns `Own<T>` where T is inferred
- Implicit conversion: `Own<T>` → `Ref<T>` for function arguments
- Error if `Ref<T>` used outside parameter position
- Track `var` modifier on parameters for mutability

## Step 4: Ownership Tracking

Add ownership analysis pass:
- Track variable states: `owned`, `moved`
- `Ref<T>` parameters = borrow (caller keeps ownership)
- `Own<T>` parameters = ownership transfer (caller loses access)
- Detect moves on assignment
- Detect use-after-move
- Detect moves in loops (error)
- Detect conditional moves (invalidate after if/else)
- Detect self-assignment (error)
- Detect overlapping moves in assignment (error)
- Left-to-right evaluation order with move tracking
- Generate drop calls at early returns (reverse declaration order)

## Step 5: Mutability Checking

Validate mutability rules:
- `val` bindings cannot be mutated or reassigned
- `var` bindings can be mutated (if fields are `var`) and reassigned
- `Ref<T>` parameters default to immutable
- `var Ref<T>` parameters allow mutation
- Borrow exclusivity: one `var Ref<T>` OR many `Ref<T>`, not both
- No self-referential assignments

## Step 6: Nullable Pointer Integration

Leverage existing nullable type system:
- `Own<T>?` is valid - nullable owned pointer
- `null` assignable to `Own<T>?`
- Safe navigation `?.` works
- Same ownership rules apply

## Step 7: Code Generation

Generate ARM64 assembly:

### Allocation
- `mmap` syscall for heap memory
- Store value at allocated address
- Return pointer

### Deallocation
- Insert `munmap` at scope exit for owned pointers
- Handle nested structs (free inner pointers first)
- Handle reassignment (free old value before new)

### Copy
- Allocate new memory
- Deep copy contents
- Recursively copy nested pointers

## Error Handling

```slang
val p = Heap.new(42)
print(p.x)                                // Error: Own<i64> has no fields

val p = Heap.new(Point{ 1, 2 })
print(p[0])                               // Error: Own<Point> not indexable

val maybeP: Own<Point>? = null
print(maybeP.x)                           // Error: use ?.x for nullable pointer

var q = Heap.new(Point{ 1, 2 })
q = q                                     // Error: cannot assign variable to itself

foo(p, p)                                 // Error: 'p' moved in first argument
```

## Quick Reference

| Expression | Type of `p` | Result Type | Notes |
|------------|-------------|-------------|-------|
| `p.x` | `Own<Struct>` or `Ref<Struct>` | field type | Auto-deref |
| `p[i]` | `Own<Array<T>>` where T is primitive | `T` | Copy of element |
| `p[i]` | `Own<Array<Own<T>>>` | `Ref<T>` | Borrows element |
| `p.field` | where field is `Own<T>` | `Ref<T>` | Auto-borrow through ref |
| `p?.x` | `Own<Struct>?` | field type? | Safe navigation |
| `p?[i]` | `Own<Array<T>>?` | `T?` | Safe navigation |
| `p == null` | `Own<T>?` | `bool` | Null check |
| `p == q` | `Own<T>` | `bool` | Identity (address) comparison |

# Alternatives

1. **Rust-style references (`&T`, `&mut T`)**: More complex borrow checker required. Rejected for MVP simplicity.

2. **C-style pointers (`*T`, `*p`)**: Less explicit, potential confusion with multiplication. The `Own<T>`/`Ref<T>` syntax is clearer and more consistent with generic types.

3. **Three pointer types (`ptr`, `ref`, `own`)**: Considered but rejected. Using `val`/`var` for mutability keeps ownership and mutability orthogonal, reuses familiar keywords.

4. **`mut` keyword for mutable borrows**: Considered `mut Ref<T>` but rejected. Using `var` is consistent with variable declarations.

5. **Implicit dereference everywhere**: Would hide pointer semantics too much. The `.deref()` requirement makes pointer operations explicit while auto-deref provides convenience where it's unambiguous.

6. **Manual memory management from start**: Would complicate MVP. Starting with allocation-only allows proving the design before adding deallocation complexity.

7. **`ptr::new(value)` syntax**: Simpler but doesn't allow for custom allocators. The `Heap.new(value)` design anticipates the allocator interface pattern.

# Future Work: Allocator Interface

The `Heap` type is designed to be the first implementation of an allocator interface. Future work could include:

```slang
// Future: Allocator interface (requires interface/trait system)
Allocator = interface {
    new<T>(value: T) -> Own<T>
    alloc<T>() -> Own<T>              // uninitialized allocation
    free<T>(p: Own<T>)                // explicit deallocation
}

// Built-in implementations
Heap: Allocator        // System heap (mmap/brk)
Arena: Allocator       // Arena/bump allocator
Pool<T>: Allocator     // Fixed-size pool allocator

// Usage with custom allocator
createWithAllocator = (alloc: Allocator, x: i64, y: i64) -> Own<Point> {
    alloc.new(Point{ x, y })
}

main = () {
    var arena = Arena.create(1024)    // 1KB arena
    val p1 = arena.new(Point{ 1, 2 })
    val p2 = arena.new(Point{ 3, 4 })
    arena.freeAll()                   // free entire arena at once
}
```

This is explicitly out of scope for the initial implementation but informs the design choice of `Heap.new()` over `Own.new()`.

# Testing

## Lexer/Parser Tests
- Token recognition for `Own`, `Ref`, `Heap`
- `Own<T>` and `Ref<T>` type parsing
- `Heap.new(expr)` expressions
- `.copy()` method calls
- `var` modifier on function parameters

## Semantic Tests
- Type inference for pointer types
- `Heap` type checking
- Auto-dereference rules
- Auto-borrowing from `Own<T>` to `Ref<T>`
- Nullable pointer rules
- `Ref<T>` only allowed in parameter position
- `var` enables mutation on parameters
- Pointer comparison (`==`) is identity (address) comparison
- Nullable pointer comparison to `null`
- Array indexing returns copy for primitives, borrow for `Own<T>`
- `Own<T?>` is invalid (T must be non-nullable)
- Safe navigation `?.` on non-nullable is error
- Array of nullable pointers: indexing returns nullable borrow
- Array literal type inference with null elements

## Ownership Tests
- Move on assignment (`val q = p` invalidates `p`)
- Use-after-move detection
- `Ref<T>` parameters borrow (caller keeps ownership)
- `Own<T>` parameters take ownership (caller loses access)
- Cannot mutate through immutable `Ref<T>`
- `var Ref<T>` allows mutation
- Mutable borrow requires `var` source (cannot create `var Ref` from `val` binding)
- Borrow exclusivity (one `var Ref<T>` OR many `Ref<T>`, not both)
- Mixed borrow + move in same call is error
- Nested ownership (struct containing `Own<T>` fields)
- Field access through `Ref` auto-borrows `Own<T>` fields
- Chained field access propagates mutability
- Closures capturing `Own<T>` move ownership into closure
- Closures cannot capture `Ref<T>` (error)
- Generics with `Ref<T>` in field position (error)
- Temporary lifetimes extend to end of statement
- Loop iteration borrows container
- Mutable iteration produces `var Ref<T>` for `var` arrays
- Cannot reassign while borrowed (including during iteration)
- Self-assignment is error
- Overlapping moves in assignment is error
- Left-to-right evaluation order with move tracking (borrows are fine)
- Early return drops owned values in reverse order
- Nullable pointer smart cast inside null check
- Smart cast in while loop condition and body
- Nested null checks with auto-borrow
- Operators auto-dereference `Own<primitive>`
- Short-circuit operators with moves (conditional move rule)
- Conditional expressions with moves (both branches invalidated)
- `break` and `continue` drop loop-local owned values
- Implicit return moves same as explicit return
- Pass-through ownership (take and return same value)
- `null` as argument to nullable parameters

## Mutability Tests
- `val` binding prevents mutation
- `var` binding allows mutation of `var` fields
- `val` field prevents mutation regardless of binding
- `var` field allows mutation with `var` binding

## Codegen Tests
- Correct assembly for allocation (`mmap`)
- Dereference loads correct address
- Auto-deallocation at scope exit
- Nested deallocation order
- `.copy()` deep copies correctly

## E2E Tests
Full programs using pointers with expected outputs:
- Basic allocation
- Stack to heap promotion (`Heap.new(stackValue)`)
- Pointers to structs with field access
- Pointers to arrays with indexing (primitives return copy)
- Pointers to arrays with indexing (owned elements return borrow)
- Nested pointers
- Nullable pointers with null checks
- Safe navigation on nullable pointers
- Linked list creation and traversal
- Ownership transfer with `Own<T>` parameters
- Auto-borrowing with `Ref<T>` and `var Ref<T>` parameters
- Memory freed correctly (no leaks in simple cases)
- Pointer identity comparison (`p == q`)
- Closures capturing owned pointers
- Loop iteration over pointer arrays
- Temporary lifetime in chained expressions
- Method calls with different receiver types (SEP 7)
- Method chaining with consuming methods
- Operators auto-dereference Own<primitive>
- Error cases:
  - Self-assignment
  - Double move in expression
  - Overlapping move paths
  - Mixed borrow types (mutable + immutable)
  - Mixed borrow and move in same call
  - Use after move
  - Mutable borrow from val binding
  - Reassign while borrowed
  - Safe navigation on non-nullable pointer
  - `Own<T?>` type declaration
  - Move in short-circuit operator (conditional move)
  - Move in conditional expression (both branches)
  - `.copy()` on stack-allocated value

# Code Examples

## Example 1: Basic Allocation

Demonstrates basic pointer allocation with `Heap.new`.

```slang
main = () {
    val p = Heap.new(42)

    // For primitives, pass to function that takes the value
    printValue = (x: i64) { print(x) }

    // Auto-borrow when passing to Ref parameter
    printRef = (r: Ref<i64>) { /* can read r */ }
    printRef(p)                           // auto-borrows p

    // Use .copy() to get an independent owned copy
    val q = p.copy()
}
```

## Example 2: Stack to Heap Promotion

Shows moving a stack-allocated value to the heap.

```slang
Point = struct {
    val x: i64
    val y: i64
}

main = () {
    // Create on stack
    val p = Point{ 10, 20 }
    print(p.x)                            // prints: 10

    // Move to heap
    val h = Heap.new(p)                   // p moved to heap
    // print(p.x)                         // Error: p was moved

    print(h.x)                            // prints: 10 (via heap pointer)

    // Useful for storing in data structures
    val points: Array<Own<Point>> = [
        Heap.new(Point{ 1, 2 }),
        Heap.new(Point{ 3, 4 })
    ]
}
```

## Example 3: Struct with Auto-Dereference

Shows auto-dereference for field access on a pointer to a struct.

```slang
Point = struct {
    val x: i64
    val y: i64
}

main = () {
    val p = Heap.new(Point{ 10, 20 })
    print(p.x)                            // prints: 10 (auto-deref)
    print(p.y)                            // prints: 20 (auto-deref)
    print(p.x + p.y)                      // prints: 30

    // Copy to get independent owned value
    val q = p.copy()
    print(q.x)                            // prints: 10
}
```

## Example 4: Array with Auto-Dereference

Shows auto-dereference for index access on a pointer to an array.

```slang
main = () {
    val arr = Heap.new([1, 2, 3, 4, 5])
    print(arr[0])                         // prints: 1 (auto-deref)
    print(arr[2])                         // prints: 3
    print(arr[0] + arr[4])                // prints: 6
    print(len(arr))                       // prints: 5 (auto-deref for len)
}
```

## Example 5: Linked List with Ownership

Demonstrates linked list with ownership transfer. Each node owns its `next` pointer.

```slang
Node = struct {
    val value: i64
    var next: Own<Node>?                  // nullable, owned by this node
}

main = () {
    // Build list from tail to head
    // Assignment moves ownership automatically
    val n3 = Heap.new(Node{ 30, null })
    val n2 = Heap.new(Node{ 20, n3 })     // n3 moves into n2.next
    val n1 = Heap.new(Node{ 10, n2 })     // n2 moves into n1.next

    // n3 and n2 are now invalid (moved)
    // print(n3.value)                    // Error: n3 was moved

    // Access through n1
    print(n1.value)                       // prints: 10
    print(n1.next?.value)                 // prints: 20
    print(n1.next?.next?.value)           // prints: 30
}                                         // n1 freed, recursively frees all nodes
```

## Example 6: Function Returning Pointer

Shows a function that allocates and returns a pointer. Returns `Own<T>` to transfer ownership.

```slang
Point = struct {
    val x: i64
    val y: i64
}

createPoint = (x: i64, y: i64) -> Own<Point> {
    Heap.new(Point{ x, y })
}

main = () {
    val p = createPoint(5, 10)            // main now owns p
    print(p.x + p.y)                      // prints: 15
}
```

## Example 7: Nullable Pointer with Conditional Logic

Shows nullable pointer handling with null checks.

```slang
Person = struct {
    val name: string
    val age: i64
    var spouse: Own<Person>?              // may or may not have a spouse
}

main = () {
    val alice = Heap.new(Person{ "Alice", 30, null })
    val bob = Heap.new(Person{ "Bob", 32, alice })  // alice moves into bob.spouse

    // Safe navigation returns nullable
    val spouseName: string? = bob.spouse?.name
    print(spouseName)                     // prints: Alice

    // Null check before access
    if (alice.spouse != null) {
        print(alice.spouse?.name)
    } else {
        print("Alice has no spouse")      // prints this
    }
}
```

## Example 8: Borrowing with Ref<T> and Ownership with Own<T>

Shows the difference between borrowing (`Ref<T>`) and ownership transfer (`Own<T>`).

```slang
Point = struct {
    var x: i64
    var y: i64
}

// Immutable borrow - cannot modify
printPoint = (p: Ref<Point>) {
    print(p.x)
    print(p.y)
    // p.x = 100                          // Error: p is not var
}

// Mutable borrow - can modify, caller keeps ownership
scalePoint = (var p: Ref<Point>, factor: i64) {
    p.x = p.x * factor
    p.y = p.y * factor
}

// Takes ownership - caller loses access
consume = (p: Own<Point>) {
    print(p.x)
}                                         // p freed here

main = () {
    var p = Heap.new(Point{ 10, 20 })

    printPoint(p)                         // borrows as Ref<Point>
    printPoint(p)                         // can borrow multiple times

    scalePoint(p, 2)                      // borrows as var Ref<Point>
    print(p.x)                            // prints: 20 (was 10 * 2)

    val q = p.copy()                      // create independent copy
    val r = p                             // moves p to r
    // print(p.x)                         // Error: p was moved

    print(q.x)                            // OK: q is independent copy (20)
    print(r.x)                            // OK: r now owns it (20)

    consume(q)                            // q ownership transferred
    // print(q.x)                         // Error: q was moved
}
```

## Example 9: Tree Structure with Ownership

Shows ownership in a binary tree structure.

```slang
TreeNode = struct {
    val value: i64
    var left: Own<TreeNode>?
    var right: Own<TreeNode>?
}

// Creates a new leaf node
leaf = (value: i64) -> Own<TreeNode> {
    Heap.new(TreeNode{ value, null, null })
}

// Creates a new internal node - children move into the node
node = (value: i64, left: Own<TreeNode>, right: Own<TreeNode>) -> Own<TreeNode> {
    Heap.new(TreeNode{ value, left, right })  // left and right move in
}

// Borrows tree to compute sum (Ref<T> = immutable borrow)
sum = (t: Ref<TreeNode>) -> i64 {
    var total = t.value
    if (t.left != null) {
        total = total + sum(t.left)       // recursive borrow
    }
    if (t.right != null) {
        total = total + sum(t.right)
    }
    total
}

main = () {
    // Build tree:      5
    //                 / \
    //                3   7
    val tree = node(5, leaf(3), leaf(7))

    print(sum(tree))                      // prints: 15 (borrows tree)
    print(sum(tree))                      // can borrow again

}                                         // tree freed, recursively frees children
```

## Example 10: Mutable Struct Fields

Shows interaction between binding mutability and field mutability.

```slang
Point = struct {
    val x: i64      // immutable field
    var y: i64      // mutable field
}

main = () {
    // val binding - nothing can be mutated
    val p = Heap.new(Point{ 1, 2 })
    // p.x = 10                           // Error: x is val field
    // p.y = 20                           // Error: p is val binding

    // var binding - only var fields can be mutated
    var q = Heap.new(Point{ 1, 2 })
    // q.x = 10                           // Error: x is val field
    q.y = 20                              // OK: q is var and y is var

    print(q.y)                            // prints: 20
}
```

## Example 11: Method-Style Ownership Transfer

Shows using `Own<T>` parameters to transfer ownership into struct fields.

```slang
Node = struct {
    val value: i64
    var next: Own<Node>?
}

// Takes ownership of 'next' and stores it in the node
setNext = (var node: Ref<Node>, next: Own<Node>) {
    node.next = next                      // OK: we own 'next', can move it
}

main = () {
    var n1 = Heap.new(Node{ 10, null })
    val n2 = Heap.new(Node{ 20, null })

    setNext(n1, n2)                       // n2 ownership transferred
    // print(n2.value)                    // Error: n2 was moved

    print(n1.next?.value)                 // prints: 20
}
```

## Example 12: Pointer Comparison

Shows identity vs value comparison for pointers.

```slang
Point = struct {
    val x: i64
    val y: i64
}

main = () {
    val p = Heap.new(Point{ 1, 2 })
    val q = Heap.new(Point{ 1, 2 })       // same values, different allocation
    val r = p.copy()                       // copy of p, different allocation

    // Identity comparison (address)
    print(p == q)                          // prints: false
    print(p == r)                          // prints: false

    // Value comparison (compare fields directly)
    print(p.x == q.x && p.y == q.y)        // prints: true
    print(p.x == r.x && p.y == r.y)        // prints: true

    // Nullable pointer comparison
    val n: Own<Point>? = null
    print(n == null)                       // prints: true
}
```

## Example 13: Closures Capturing Pointers

Shows how closures interact with pointer ownership.

```slang
Point = struct {
    val x: i64
    val y: i64
}

main = () {
    val p = Heap.new(Point{ 10, 20 })

    // Closure captures p, moving ownership into the closure
    val printX = () {
        print(p.x)                         // closure owns p
    }

    // print(p.x)                          // Error: p was moved into closure

    printX()                               // prints: 10
    printX()                               // prints: 10 (can call multiple times)
}                                          // p freed when closure goes out of scope
```

## Example 14: Loop Iteration with Pointers

Shows borrowing semantics during iteration.

```slang
Point = struct {
    val x: i64
    val y: i64
}

main = () {
    val points = Heap.new([
        Heap.new(Point{ 1, 2 }),
        Heap.new(Point{ 3, 4 }),
        Heap.new(Point{ 5, 6 })
    ])

    // Iteration borrows the array; each p borrows the element
    for p in points {
        print(p.x)                         // p: Ref<Point>
    }
    // prints: 1, 3, 5

    // Array still owns all points after iteration
    print(points[0].x)                     // prints: 1

    // Sum all x values
    var sum = 0
    for p in points {
        sum = sum + p.x
    }
    print(sum)                             // prints: 9
}
```

## Example 15: Temporary Lifetimes

Shows that temporaries live until the end of the statement.

```slang
Point = struct {
    val x: i64
    val y: i64
}

createPoint = (x: i64, y: i64) -> Own<Point> {
    Heap.new(Point{ x, y })
}

main = () {
    // Temporary lives for the entire statement
    val sum = createPoint(10, 20).x + createPoint(30, 40).y
    print(sum)                             // prints: 50
    // Both temporaries freed here

    // Chained access on temporary
    print(createPoint(5, 10).x)            // prints: 5
    // Temporary freed here

    // Nested temporaries
    Outer = struct {
        val inner: Own<Point>
    }

    createOuter = () -> Own<Outer> {
        Heap.new(Outer{ Heap.new(Point{ 100, 200 }) })
    }

    val v = createOuter().inner.x          // OK: all temps live until semicolon
    print(v)                               // prints: 100
}
```

## Example 16: Array Indexing Semantics

Shows how array indexing returns copies for primitives but borrows for owned pointers.

```slang
Point = struct {
    val x: i64
    val y: i64
}

main = () {
    // Primitive array - indexing returns copy
    val nums = Heap.new([10, 20, 30])
    val x = nums[0]                        // x: i64 (copy of value)
    val y = nums[0]                        // y: i64 (another copy)
    print(x + y)                           // prints: 20

    // Owned pointer array - indexing borrows
    val points: Own<Array<Own<Point>>> = Heap.new([
        Heap.new(Point{ 1, 2 }),
        Heap.new(Point{ 3, 4 }),
        Heap.new(Point{ 5, 6 })
    ])

    val p = points[0]                      // p: Ref<Point> (borrow)
    print(p.x)                             // prints: 1

    val q = points[0]                      // q: Ref<Point> (can borrow again)
    print(q.x)                             // prints: 1 (same element)

    // Array still owns all elements
    print(points[0].x)                     // prints: 1

    // To replace an element, use .set()
    val old = points.set(0, Heap.new(Point{ 100, 200 }))
    // old: Own<Point> (the replaced element)
    print(points[0].x)                     // prints: 100

    // To remove an element, use .remove()
    val removed = points.remove(0)         // removed: Own<Point>, shifts elements
    print(len(points))                     // prints: 2
}
```

## Example 17: Compile-Time Safety Checks

Shows various compile-time errors that prevent undefined behavior.

```slang
Point = struct {
    var x: i64
    var y: i64
}

// ❌ Error: self-assignment
bad1 = () {
    var p = Heap.new(Point{ 1, 2 })
    p = p                                  // Error: cannot assign variable to itself
}

// ❌ Error: double move in expression
bad2 = () {
    val p = Heap.new(Point{ 1, 2 })
    consume(p, p)                          // Error: 'p' moved twice
}

// ❌ Error: use after move in expression
bad3 = () {
    val p = Heap.new(Point{ 1, 2 })
    foo(take(p), p.x)                      // Error: 'p' used after move
}

// ❌ Error: overlapping moves
Container = struct {
    var data: Own<Point>
}

bad4 = () {
    var c = Heap.new(Container{ Heap.new(Point{ 1, 2 }) })
    c = c.data                             // Error: overlapping move paths
}

// ❌ Error: mixed borrow types
bad5 = () {
    var p = Heap.new(Point{ 1, 2 })
    mixedBorrow(p, p)                      // Error: mutable + immutable borrow
}

mixedBorrow = (var a: Ref<Point>, b: Ref<Point>) {
    a.x = b.x + 1
}

// ❌ Error: mixed borrow and move
bad6 = () {
    val p = Heap.new(Point{ 1, 2 })
    borrowAndMove(p, p)                    // Error: cannot borrow and move same value
}

borrowAndMove = (r: Ref<Point>, p: Own<Point>) {
    print(r.x)
}

// ❌ Error: mutable borrow from val binding
bad7 = () {
    val p = Heap.new(Point{ 1, 2 })        // val binding
    mutate(p)                              // Error: cannot create mutable borrow from val
}

mutate = (var p: Ref<Point>) {
    p.x = 10
}

// ❌ Error: reassign while borrowed
bad8 = () {
    var arr = Heap.new([1, 2, 3])
    for x in arr {
        arr = Heap.new([4, 5, 6])          // Error: cannot reassign while borrowed
    }
}

// ✅ OK: operators auto-dereference Own<primitive>
good0 = () {
    val p = Heap.new(42)
    val sum = p + 1                        // OK: auto-derefs, sum is 43
    val cmp = p > 10                       // OK: auto-derefs, cmp is true
}

// ✅ OK: multiple immutable borrows
good1 = () {
    val p = Heap.new(Point{ 1, 2 })
    readBoth(p, p)                         // OK: both immutable
}

readBoth = (a: Ref<Point>, b: Ref<Point>) {
    print(a.x + b.x)
}

// ✅ OK: nullable smart cast
good2 = () {
    val p: Own<Point>? = maybeGet()
    if (p != null) {
        print(p.x)                         // OK: smart cast to Own<Point>
    }
}
```

## Example 18: Nullable Pointer Smart Cast

Shows smart casting of nullable pointers inside null checks.

```slang
Point = struct {
    var x: i64
    var y: i64
}

main = () {
    val p: Own<Point>? = maybeGetPoint()

    // Without null check - must use ?.
    print(p?.x)                            // OK: safe navigation

    // With null check - smart cast to non-null
    if (p != null) {
        print(p.x)                         // OK: p is Own<Point> here
        print(p.y)                         // OK: no ?. needed

        // Ownership still applies
        val q = p                          // moves p
        // print(p.x)                      // Error: p was moved
    }

    // After if: conditional move applies
    // print(p?.x)                         // Error: p may have been moved

    // Reassignment vs move
    var r: Own<Point>? = maybeGetPoint()
    if (r != null) {
        r = null                           // reassign, not move
    }
    print(r?.x)                            // OK: r wasn't moved
}
```

## Example 19: Field Access Through References

Shows how field access works when accessing through borrowed references.

```slang
Inner = struct {
    var value: i64
}

Outer = struct {
    val inner: Own<Inner>
}

Container = struct {
    val outer: Own<Outer>
}

// Immutable access - all borrows are immutable
readDeep = (c: Ref<Container>) {
    // c is Ref<Container>
    // c.outer is Ref<Outer> (auto-borrow of Own<Outer> field)
    // c.outer.inner is Ref<Inner> (auto-borrow of Own<Inner> field)
    val v = c.outer.inner.value            // v: i64 (copy of primitive)
    print(v)
}

// Mutable access - mutability propagates through chain
mutateDeep = (var c: Ref<Container>) {
    // c is var Ref<Container>
    // c.outer is var Ref<Outer> (inherits var)
    // c.outer.inner is var Ref<Inner> (inherits var)
    c.outer.inner.value = 100              // OK: can mutate through var chain
}

main = () {
    var container = Heap.new(Container{
        Heap.new(Outer{
            Heap.new(Inner{ 42 })
        })
    })

    readDeep(container)                    // prints: 42
    mutateDeep(container)                  // modifies to 100
    readDeep(container)                    // prints: 100
}
```
