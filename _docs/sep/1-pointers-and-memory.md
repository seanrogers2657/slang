# Status

IMPLEMENTED, 2026-01-05

## Implementation Status

| Feature | Status | Notes |
|---------|--------|-------|
| `*T` type | ✅ Done | Ownership tracking, auto-free on scope exit |
| `&T` type | ✅ Done | Immutable borrow, parameter-only |
| `&&T` type | ✅ Done | Mutable borrow, parameter-only |
| `Heap.new(T)` | ✅ Done | Bump allocator with size-class free lists |
| Move semantics | ✅ Done | Use-after-move detection, conditional moves |
| Auto-borrowing | ✅ Done | `*T` → `&T`/`&&T` at call sites |
| Borrow exclusivity | ✅ Done | Multiple `&T` OR one `&&T` per call |
| `*T?` nullable | ✅ Done | Nullable owned pointers |
| `?.` safe call | ✅ Done | Works through `*T?`, `&T?`, `&&T?` |
| T → T? coercion | ✅ Done | `*T` assignable to `*T?` parameters |
| Ownership restore | ✅ Done | `x = fn(x)` pattern restores ownership |
| Bump allocator | ✅ Done | 1MB arenas, O(1) alloc/free, ~200x memory savings |
| `.copy()` | ⏳ Pending | Deep copy not yet implemented |
| `Heap.shrink()` | ⏳ Future | Return empty arenas to OS |
| `Weak<T>` | ⏳ Future | Weak references for cycles |
| `Rc<T>` | ⏳ Future | Reference counting for shared ownership |
| `Heap.tryNew` | ⏳ Future | Fallible allocation |

# Summary/Motivation

Add heap allocation through pointer types to Slang, enabling dynamic memory allocation and data structures like linked lists and trees. This introduces three pointer types:

- **`*T`** - Owned pointer. You control the lifetime; memory is freed when the owner goes out of scope.
- **`&T`** - Immutable borrowed reference. Read-only access to someone else's data.
- **`&&T`** - Mutable borrowed reference. Can mutate `var` fields of borrowed data.

**Key principle**: `val`/`var` controls **reassignability** only; `&T` vs `&&T` controls **borrow mutability**:

- **`val`** - Cannot reassign the binding. Can still mutate `var` fields through the pointer.
- **`var`** - Can reassign the binding.

This keeps ownership and mutability as orthogonal concepts, reusing keywords users already understand.

`Heap` is a built-in allocator type with a `new` method that returns `*T`. This design allows for future extensibility - custom allocators can implement the same interface.

The ownership model is simple:
- **Assignment moves** - `val q = p` transfers ownership, making `p` invalid
- **Function parameters borrow or take ownership** - `&T` borrows, `*T` takes ownership

Slang already has nullable types (`T?`) and arrays (`Array<T>`). This SEP builds on those foundations - nullable pointers (`*T?`) provide a natural way to represent optional references (e.g., linked list terminators).

# Goals/Non-Goals

## Allocation
- [goal] `Heap` built-in allocator type with `new(value)` method
- [goal] `Heap.new(value)` returns `*T` (owned pointer)
- [goal] Type inference for pointer types

## Pointer Types
- [goal] `*T` - owned pointer, controls lifetime of pointed-to memory
- [goal] `&T` - borrowed reference, temporary access without ownership
- [goal] Automatic borrowing from `*T` to `&T` based on context (no explicit `.borrow()`)
- [goal] Auto-dereference for field access (`p.field`) and array indexing (`p[i]`)
- [goal] Nullable pointers via existing `T?` syntax (`*T?` with `null`)

## Mutability
- [goal] `val`/`var` controls reassignability only (can you point to something else?)
- [goal] `&T` vs `&&T` controls borrow mutability (can you mutate through the reference?)
- [goal] `val p: *T` - cannot reassign p, CAN mutate var fields through p
- [goal] `var p: *T` - can reassign p, can mutate var fields through p
- [goal] `p: &T` - immutable borrow (read-only access)
- [goal] `p: &&T` - mutable borrow (can mutate var fields)

## Ownership & Memory Safety
- [goal] Single ownership model - each allocation has exactly one owner
- [goal] `&T` parameters borrow (caller keeps ownership)
- [goal] `*T` parameters take ownership (caller loses access)
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
- [non-goal] Fallible allocation (`Heap.tryNew() -> *T?`) - future work
- [non-goal] Custom allocator interface - future work, but design anticipates it
- [non-goal] Address-of operator (`&variable`) - can only get pointers via `Heap.new`
- [non-goal] Pointer arithmetic
- [non-goal] Garbage collection or reference counting
- [non-goal] Explicit lifetime annotations - ownership rules avoid need for these
- [non-goal] Partial moves - cannot move individual struct fields

# APIs

## Allocator

- `Heap` - Built-in allocator type for heap memory allocation.
- `Heap.new(value)` - Allocates memory on the heap, stores the value, and returns `*T`. Type T is inferred from the value. The value can be:
  - A literal: `Heap.new(Point{ 1, 2 })`
  - A stack-allocated variable: `Heap.new(p)` (moves `p` to heap)
  - An owned pointer: `Heap.new(ownedPtr)` (creates `**T` - a pointer to a pointer)
- **Allocation failure:** `Heap.new()` panics if allocation fails (out of memory). This matches Go, Swift, and Kotlin behavior. For fallible allocation, see Future Work (`Heap.tryNew`).

## Pointer Types

- `*T` - Owned pointer. You control the lifetime; memory freed when owner goes out of scope.
- `&T` - Immutable borrowed reference. Read-only temporary access without ownership.
- `&&T` - Mutable borrowed reference. Can mutate `var` fields of borrowed data.
- `*T?` - Nullable owned pointer (can be `null` or a valid pointer).
- `.copy()` - Create a deep copy, returning a new independent `*T`.

## Automatic Borrowing

Borrowing happens automatically based on context - no explicit `.borrow()` method needed:

| Context | Source | Result |
|---------|--------|--------|
| Pass `*T` to param `&T` | `foo(p)` | Auto-borrow (immutable) |
| Pass `*T` to param `&&T` | `bar(p)` | Auto-borrow (mutable) |
| Method call with `self: &T` | `p.method()` | Auto-borrow based on receiver |
| Field access on `*T` | `p.field` | Auto-dereference |

## Array Indexing

Array indexing behavior depends on element type:

- **Primitive elements** (`Array<i64>`, etc.): `arr[i]` returns a copy of the value
- **Owned pointer elements** (`Array<*T>`): `arr[i]` returns `&T` or `&&T` (borrows the element)
- **Mutability**: Use explicit `&&T` parameter to get mutable access to elements

```slang
// Primitive array - indexing returns copy
val nums = Heap.new([1, 2, 3])
val x = nums[0]               // x: i64 (copy)

// Owned pointer array - indexing borrows
val points: *Array<*Point> = Heap.new([...])
val p = points[0]             // p: &Point (immutable borrow)
// points[0] still owns the element

// Mutable access to array elements
mutateFirst = (arr: &&Array<*Point>) {
    val x = arr[0]            // x: &&Point (mutable borrow)
    x.x = 100                 // OK: can mutate through &&T
}

// To actually remove/replace elements, use methods:
val removed = points.remove(0)    // Returns *Point, shifts elements
val old = points.set(0, newPoint) // Replaces element, returns old one
```

**Rationale:** Moving elements out of arrays via indexing would leave "holes" and require complex tracking. Borrowing is always safe.

## Field Access Through &T

When accessing an `*T` field through a `&T` or `&&T`, the result is automatically borrowed:

- Through `&Container`: field `*T` becomes `&T`
- Through `&&Container`: field `*T` becomes `&&T`

```slang
Container = struct {
    val data: *Point
}

// Immutable access
readData = (c: &Container) {
    val p = c.data            // p: &Point (auto-borrow)
    print(p.x)                // OK: read access
    // p.x = 10               // Error: &T is read-only
}

// Mutable access
mutateData = (c: &&Container) {
    val p = c.data            // p: &&Point (auto-borrow, inherits mutability)
    p.x = 10                  // OK: can mutate through &&T
}
```

**Chained access:** Mutability propagates through the chain.

```slang
Outer = struct { val inner: *Inner }
Inner = struct { val point: *Point }

deepAccess = (o: &&Outer) {
    val p = o.inner.point     // p: &&Point
    // o is &&Outer
    // o.inner is &&Inner (auto-borrow, inherits mutability)
    // o.inner.point is &&Point (auto-borrow, inherits mutability)
    p.x = 100                 // OK
}
```

**Rationale:** You cannot move out of borrowed data. Auto-borrowing is the only safe option.

## Mutability

**Key principle**: `val`/`var` controls **reassignability** only; `&T` vs `&&T` controls **borrow mutability**:

| Declaration | Can Reassign | Can Mutate Fields |
|-------------|--------------|-------------------|
| `val p: *T` | No | Yes (if field is `var`) |
| `var p: *T` | Yes | Yes (if field is `var`) |
| `p: &T` | — | No |
| `p: &&T` | — | Yes (if field is `var`) |

## Type Usage

- `*T` - Can be used anywhere: variables, parameters, return types, struct fields. T must be a non-nullable type.
- `&T` - Can be used as function parameters. Cannot be stored in struct fields or returned from functions (would dangle). T must be a non-nullable type.
- `*T?` - Nullable owned pointer. The pointer may be null; if non-null, it points to a valid T.
- `&T?` - Nullable borrowed reference. Can be used as function parameter for optional borrowed data.
- **Recursive types** - Structs can reference themselves via `*Self?` fields (must be nullable to allow a base case).
- **No `*T?`** - Pointers to nullable types are not supported. Use `*T?` instead.

```slang
// ✅ Valid pointer types
val p: *Point = Heap.new(Point{ 1, 2 })
val q: *Point? = null
val r: *Array<*Point> = Heap.new([...])

// ❌ Invalid: T? inside *
val bad: *Point? = ...             // Error: *T requires non-nullable T
```

```slang
// Recursive type example
Node = struct {
    val value: i64
    var next: *Node?               // nullable self-reference
}

// Optional borrowed data
maybeUse = (p: &Point?) {
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
- **Move-only types:** `*T` and any struct containing `*T` fields

```slang
// Copyable - all fields are primitives
Point = struct { val x: i64; val y: i64 }
val p1 = Point{ 1, 2 }
val p2 = p1            // Copy - both valid

// Move-only - contains *T
Container = struct { val data: *Point }
val c1 = Container{ Heap.new(Point{ 1, 2 }) }
val c2 = c1            // Move - c1 is invalid

// To copy move-only types, use .copy()
val c3 = c1.copy()     // Deep copy - both valid (if c1 wasn't moved)
```

This affects array indexing: `arr[i]` returns a copy for copyable element types, but `&T` for move-only element types.

## Auto-Dereference

- Field access - `p.field` automatically dereferences to access struct fields.
- Index access - `p[i]` automatically dereferences to access array elements.
- Safe navigation - `?.` works with nullable pointers just like other nullable types.
- **Error on non-nullable:** Using `?.` on a non-nullable pointer is a compile error.

```slang
val p = Heap.new(Point{ 1, 2 })       // *Point, not nullable
print(p?.x)                            // Error: safe navigation on non-nullable type

val q: *Point? = maybeGet()
print(q?.x)                            // OK: q is nullable
```

## Comparison

- `p == q` for pointers is **identity comparison** (same address), not value comparison.
- For value comparison, compare fields directly: `p.x == q.x && p.y == q.y`.
- Nullable pointers can be compared to `null`: `p == null`.
- `*T` can be compared with `&T` - both are identity (address) comparison.

```slang
val p = Heap.new(Point{ 1, 2 })
val q = Heap.new(Point{ 1, 2 })
val r = p.copy()

print(p == q)                         // false: different allocations
print(p == r)                         // false: copy is separate allocation
print(p.x == q.x && p.y == q.y)       // true: same field values

val n: *Point? = null
print(n == null)                      // true

// *T vs &T comparison
compare = (r: &Point) -> bool {
    r == p                            // OK: identity comparison
}
print(compare(p))                     // true: same address
print(compare(q))                     // false: different address
```

## Implicit Conversions

- `*T` → `&T` - Automatic when passing to function expecting `&T` parameter.
- `*T?` → `&T` - **Not allowed.** Must unwrap first via null check or smart cast.

```slang
foo = (p: &Point) { print(p.x) }

main = () {
    val p: *Point? = maybeGet()

    // foo(p)                         // Error: cannot auto-borrow nullable pointer

    if (p != null) {
        foo(p)                         // OK: p is smart cast to *Point
    }
}
```

# Ownership Model

Slang uses a simple ownership model that provides memory safety without garbage collection or lifetime annotations.

**Two concepts to remember:**
1. **Ownership** - `*T` means you control the lifetime; `&T` means you're borrowing
2. **Mutability** - `val` means immutable; `var` means mutable

## Core Concepts

### Single Ownership

Every `*T` has exactly one owner. When the owner goes out of scope, the memory is automatically freed.

```slang
main = () {
    val p = Heap.new(Point{ 1, 2 })    // p: *Point, main owns it
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

### Borrowing with &T

`&T` parameters borrow - the caller keeps ownership.

```slang
// &T parameter = immutable borrow (read-only)
printPoint = (p: &Point) {
    print(p.x)
    print(p.y)
    // p.x = 100                        // Error: &T is read-only
}

// &&T parameter = mutable borrow (read-write)
scalePoint = (p: &&Point, factor: i64) {
    p.x = p.x * factor                  // OK: &&T allows mutation
    p.y = p.y * factor
}

main = () {
    var p = Heap.new(Point{ 1, 2 })

    printPoint(p)                       // borrows as &Point (immutable)
    printPoint(p)                       // can borrow again

    scalePoint(p, 10)                   // borrows as &&Point (mutable)
    print(p.x)                          // prints: 10

    print(p.x)                          // OK: main still owns p
}
```

### Ownership Transfer with *T

`*T` parameters take ownership - the caller loses access.

```slang
// *T parameter = takes ownership
consume = (p: *Point) {
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
    val h = Heap.new(p)                 // moves p to heap, h: *Point

    // print(p.x)                       // Error: p was moved
    print(h.x)                          // OK: access through h
}
```

This is useful when you create a value and later decide it needs to live on the heap (e.g., to store in a data structure).

### Deep Copy with `.copy()`

To create an independent copy (both remain valid), use `.copy()`. This is a built-in method on `*T`.

```slang
main = () {
    val p = Heap.new(Point{ 1, 2 })
    val q = p.copy()                    // deep copy, new allocation

    print(p.x)                          // OK: p still valid
    print(q.x)                          // OK: q is independent
}                                       // both freed independently
```

**Safe navigation with `.copy()`:** For nullable pointers, `?.copy()` returns `*T?`:
```slang
main = () {
    val p: *Point? = maybeGetPoint()
    val q: *Point? = p?.copy()      // q is null if p is null, otherwise deep copy
}
```

**`.copy()` is only for `*T`:** Stack-allocated copyable types use assignment to copy. Using `.copy()` on a stack value is an error:
```slang
main = () {
    val p = Point{ 1, 2 }              // Stack-allocated, copyable
    val q = p                          // Copy via assignment - both valid
    // val r = p.copy()                // Error: .copy() is only for *T

    val h = Heap.new(Point{ 1, 2 })    // Heap-allocated
    val i = h.copy()                   // OK: deep copy of owned pointer
}
```

**Nested structures:** `.copy()` performs a deep copy, recursively copying all `*T` fields.

```slang
Container = struct {
    val data: *Point
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

Functions can return `*T` - ownership transfers to the caller.

```slang
createPoint = (x: i64, y: i64) -> *Point {
    Heap.new(Point{ x, y })             // ownership transferred to caller
}

main = () {
    val p = createPoint(10, 20)         // main now owns p
    print(p.x)
}                                       // p freed here
```

## Pointers in Structs

Struct fields with `*T` types are **owned** by the struct.

```slang
Container = struct {
    val data: *Point
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
    var next: *Node?                // nullable, owned by this node
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
| Allocate literal | `Heap.new(Point{ 1, 2 })` | Returns `*T`, caller owns it |
| Stack to heap | `Heap.new(p)` | Moves stack value to heap, returns `*T` |
| Assign | `val q = p` | Moves ownership, `p` invalid |
| Copy | `p.copy()` | Deep copy, both valid |
| Pass to `&T` param | `f(p)` | Borrow (immutable) |
| Pass to `&&T` param | `f(p)` | Borrow (mutable) |
| Pass to `*T` param | `f(p)` | Transfers ownership |
| Return `*T` | `return p` | Transfers to caller |
| Reassign `var` | `p = new` | Old value auto-freed |
| Scope exit | `}` | Owner's memory freed |

## Compiler Errors

```slang
// Error: use after move
val p = Heap.new(Point{ 1, 2 })
val q = p
print(p.x)                              // Error: 'p' was moved to 'q'

// Error: cannot mutate through immutable &T
readPoint = (p: &Point) {
    p.x = 100                           // Error: p is not var
}

// Error: cannot store &T in struct
BadStruct = struct {
    val cached: &Point              // Error: &T cannot be stored
}

// Error: cannot return &T
bad = (p: &Point) -> &Point {
    p                                   // Error: cannot return &T
}

// val binding CAN mutate var fields (val only controls reassignment)
Point = struct { var x: i64; var y: i64 }
main = () {
    val p = Heap.new(Point{ 1, 2 })
    p.x = 10                            // OK: x is a var field
    // p = other                        // Error: p is val, cannot reassign
}
```

# Edge Cases & Rules

This section documents specific rules to prevent undefined behavior.

## Rule: No Self-Referential Structures

Cannot assign a pointer into a field of the same struct instance.

```slang
Node = struct {
    var next: *Node?
}

main = () {
    var n = Heap.new(Node{ null })
    n.next = n                         // Error: cannot create self-reference
}
```

**Rationale:** Would cause infinite loop or double-free during deallocation.

## Rule: `&T` Cannot Be Stored or Returned

`&T` can only appear as function parameter types. Cannot be stored in variables, returned, or used in struct fields.

```slang
// ✅ OK: &T as parameter
printPoint = (p: &Point) { print(p.x) }

// ❌ Error: &T as return type
bad1 = (p: &Point) -> &Point { p }

// ❌ Error: &T as local variable type
main = () {
    val p = Heap.new(Point{ 1, 2 })
    val borrowed: &Point = p       // Error: cannot store &T
}

// ❌ Error: &T as struct field
Cache = struct {
    val ref: &Point                // Error: &T cannot be stored
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
    val inner: *Inner
}

main = () {
    val outer = Outer{ Heap.new(Inner{ 42 }) }
    val extracted = outer.inner        // Error: cannot move field
}
```

**Rationale:** Keeps structs fully valid or fully invalid.

## Rule: Val Binding Can Create Mutable Borrows

Since `val`/`var` only controls reassignability (not mutability), a `val` binding CAN create mutable borrows.

```slang
mutate = (p: &&Point) {
    p.x = 10
}

main = () {
    val p = Heap.new(Point{ 1, 2 })   // val binding
    mutate(p)                          // OK: val only prevents reassigning p
    print(p.x)                         // prints 10

    // p = other                       // Error: cannot reassign val binding
}
```

**Rationale:** `val` means you can't reassign the binding itself. Mutating through the pointer doesn't change what `p` points to - it changes the pointed-to data. This is similar to Java's `final` or JavaScript's `const` for object references.

## Rule: Borrow Exclusivity

A value can have **either** one mutable borrow (`&&T`) **or** any number of immutable borrows (`&T`), but not both simultaneously.

```slang
// ✅ OK: multiple immutable borrows
readBoth = (a: &Point, b: &Point) {
    print(a.x + b.x)
}
main = () {
    val p = Heap.new(Point{ 1, 2 })
    readBoth(p, p)                     // OK: both are immutable borrows
}

// ❌ Error: multiple mutable borrows
bothMutate = (a: &&Point, b: &&Point) {
    a.x = 10
    b.x = 20
}
main = () {
    var p = Heap.new(Point{ 1, 2 })
    bothMutate(p, p)                   // Error: cannot have two mutable borrows
}

// ❌ Error: mutable + immutable borrow
mixedBorrow = (a: &&Point, b: &Point) {
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
    val p: *Point? = maybeGetPoint()

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
    val p: *Point? = maybeGetPoint()

    if (p != null) {
        // p is smart cast to *Point inside this block
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
    var p: *Point? = maybeGetPoint()

    if (p != null) {
        p = null                       // reassign, not move
    }

    print(p?.x)                        // OK: p wasn't moved, just reassigned
}
```

**While loops:** Smart casting also works in `while` loop bodies:
```slang
main = () {
    var p: *Point? = maybeGetPoint()

    while (p != null) {
        print(p.x)                     // p: *Point (smart cast)
        p = getNextPoint()             // may reassign to null
    }
}
```

**Nested null checks:** When accessing nullable fields on smart-casted values:
```slang
Container = struct {
    val data: *Point?
}

main = () {
    val c: *Container? = maybeGetContainer()

    if (c != null) {
        // c: *Container (smart cast)
        if (c.data != null) {
            // c.data: &Point (auto-borrow + smart cast)
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

## Rule: Closures Cannot Capture `&T`

Closures can be stored or returned, so they cannot capture `&T` (would violate "no storing &T").

```slang
// ❌ Error: cannot capture &T
useRef = (p: &Point) {
    val f = () {
        print(p.x)                     // Error: cannot capture &T
    }
}

// ✅ OK: capture *T (moves ownership into closure)
main = () {
    val p = Heap.new(Point{ 1, 2 })

    val f = () {
        print(p.x)                     // p moved into closure
    }

    // print(p.x)                      // Error: p was moved into closure
    f()                                // OK: closure owns p
}

// ✅ OK: pass &T as parameter instead of capturing
forEach = (arr: &Array<i64>, f: (i64) -> void) {
    for (var i = 0; i < len(arr); i = i + 1) {
        f(arr[i])
    }
}
```

**Rationale:** Closures can escape their creating scope; captured `&T` could dangle.

## Rule: Generics Cannot Store `&T`

Type parameters can only be instantiated with `&T` if used exclusively in function parameter positions.

```slang
// ✅ OK: *T as type argument for fields
List = struct<T> {
    var items: Array<T>
}

main = () {
    var list: List<*Point> = List{ [] }
    list.items = append(list.items, Heap.new(Point{ 1, 2 }))
}

// ❌ Error: &T as type argument for fields
Cache = struct<T> {
    val item: T                        // if T = &Point, this is invalid
}

main = () {
    val p = Heap.new(Point{ 1, 2 })
    val c: Cache<&Point> = ...     // Error: &T cannot be stored
}
```

**Rationale:** Consistent with "&T cannot be stored" rule.

## Rule: Temporary Lifetimes

Temporaries (values returned from function calls) live until the end of the statement.

```slang
createPoint = () -> *Point {
    Heap.new(Point{ 10, 20 })
}

main = () {
    // Temporary *Point lives for the full statement
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

Assigning an `*T` variable to itself is a compile-time error.

```slang
main = () {
    var p = Heap.new(Point{ 1, 2 })
    p = p                              // Error: cannot assign variable to itself
}
```

**Rationale:** Self-assignment would move the value out (invalidating `p`) before the assignment drops the old value (which was already moved), causing use-after-free.

## Rule: No Overlapping Moves in Assignment

The left-hand side and right-hand side of an assignment cannot share paths when `*T` moves are involved.

```slang
Container = struct {
    var data: *Data
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

    // Assuming consume takes *T (moves) and read takes &T (borrows)
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
mixed = (r: &Point, p: *Point) {
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

When a function returns early, all live `*T` values are dropped in reverse declaration order.

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
    val points: *Array<*Point> = Heap.new([
        Heap.new(Point{ 1, 2 }),
        Heap.new(Point{ 3, 4 })
    ])

    // x borrows each element (cannot move out of array)
    for x in points {
        print(x.x)                     // x: &Point (borrowed)
    }

    // Array still owns all points
    print(points[0].x)                 // OK
}

// Mutable iteration: use &&T to get mutable access
mutateAll = (points: &&Array<*Point>) {
    // x is &&Point when iterating through &&T
    for x in points {
        x.x = x.x * 2                  // OK: can mutate through &&T
    }
}

main = () {
    val points = Heap.new([
        Heap.new(Point{ 1, 2 }),
        Heap.new(Point{ 3, 4 })
    ])

    mutateAll(points)
    print(points[0].x)                 // prints: 2
}
```

**Rationale:** Moving elements out during iteration would leave holes. Borrowing is safe. Use `&&T` for mutable access to elements.

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

## Rule: Operators Auto-Dereference `*primitive`

Arithmetic and comparison operators auto-dereference `*T` for primitive types. The result is the primitive type, not a pointer.

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
foo = (p: *Point?) { ... }
bar = (r: &Point?) { ... }

main = () {
    foo(null)                          // OK: null is valid for *T?
    bar(null)                          // OK: null is valid for &T?
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

For `Array<*T?>`, indexing returns `&T?` (nullable borrow).

```slang
main = () {
    val arr: *Array<*Point?> = Heap.new([
        Heap.new(Point{ 1, 2 }),
        null,
        Heap.new(Point{ 3, 4 })
    ])

    val p = arr[0]                     // p: &Point? (non-null borrow)
    val q = arr[1]                     // q: &Point? (null)

    print(p?.x)                        // prints: 1
    print(q?.x)                        // prints: null
}
```

**Rationale:** The borrow inherits the nullability of the element type.

## Rule: Array Literal Type Inference

Array literals containing both `*T` and `null` infer element type `*T?`.

```slang
main = () {
    // Inferred as Array<*Point?>
    val arr = [
        Heap.new(Point{ 1, 2 }),
        null,
        Heap.new(Point{ 3, 4 })
    ]

    // Explicit type annotation works too
    val arr2: Array<*Point?> = [Heap.new(Point{ 5, 6 }), null]
}
```

**Rationale:** Consistent with nullable type inference elsewhere in the language.

## Rule: Implicit Return Moves

Implicit return (expression as last statement) moves ownership just like explicit `return`.

```slang
// These are equivalent:
createExplicit = () -> *Point {
    val p = Heap.new(Point{ 1, 2 })
    return p                           // Explicit return, moves p
}

createImplicit = () -> *Point {
    val p = Heap.new(Point{ 1, 2 })
    p                                  // Implicit return, moves p
}

// Direct allocation works too
createDirect = () -> *Point {
    Heap.new(Point{ 1, 2 })            // Implicit return of temporary
}
```

**Rationale:** Implicit and explicit returns should have identical ownership semantics.

## Rule: Pass-Through Ownership

A function can take ownership and immediately return it (pass-through).

```slang
// Valid: takes ownership, returns same value
identity = (p: *Point) -> *Point {
    p                                  // Ownership transfers through
}

// Useful for conditional wrapping
maybeWrap = (p: *Point, shouldWrap: bool) -> *Container {
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
mutate = (p: &&Point) {
    p.x = 10        // OK: can modify through &&T
}

// Without var: read-only access
read = (p: &Point) {
    print(p.x)      // OK: can read
    // p.x = 10     // Error: p is not var
}
```

**Common confusion:**
```slang
// This does NOT mean p can be reassigned inside the function
// It means you can mutate the data p points to
mutate = (p: &&Point) {
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
    var next: *Node?
    var prev: *Node?       // Error: would need two owners
}

// ❌ Tree with parent pointer
TreeNode = struct {
    var children: Array<*TreeNode>
    var parent: *TreeNode? // Error: parent owns child, child can't own parent
}

// ❌ Graph with cycles
GraphNode = struct {
    var neighbors: Array<*GraphNode>   // Error: cycles = multiple owners
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
    var nodes: *Array<DLLNode>
    var head: i64
    var tail: i64
}

// Helper to get node by index
getNode = (list: &DoublyLinkedList, idx: i64) -> &DLLNode {
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
    var children: Array<*TreeNode>
    // No parent field - passed during traversal
}

// Parent available as parameter, not stored
traverseWithParent = (
    node: &TreeNode,
    parent: &TreeNode?,
    visit: (&TreeNode, &TreeNode?) -> void
) {
    visit(node, parent)

    for (var i = 0; i < len(node.children); i = i + 1) {
        traverseWithParent(node.children[i], node, visit)
    }
}

// Find path to root by walking up via recursion
findDepth = (node: &TreeNode, parent: &TreeNode?) -> i64 {
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
neighbors = (g: &Graph, nodeIdx: i64) -> Array<i64> {
    var result: Array<i64> = []
    for (var i = 0; i < len(g.edges); i = i + 1) {
        if (g.edges[i].from == nodeIdx) {
            result = append(result, g.edges[i].to)
        }
    }
    result
}

// BFS traversal
bfs = (g: &Graph, start: i64) {
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
newGraph = () -> *Graph {
    Heap.new(Graph{ [], [], [] })
}

// Add node, returns its index (reuses deleted slots)
addNode = (g: &&Graph, label: string, weight: i64) -> i64 {
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
addEdge = (g: &&Graph, from: i64, to: i64) {
    g.edges = append(g.edges, Edge{ from, to, false })
}

// Add undirected edge (two directed edges)
connect = (g: &&Graph, a: i64, b: i64) {
    addEdge(g, a, b)
    addEdge(g, b, a)
}

// Delete a node (marks as deleted, adds to free list)
deleteNode = (g: &&Graph, nodeIdx: i64) {
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
deleteEdge = (g: &&Graph, from: i64, to: i64) {
    for (var i = 0; i < len(g.edges); i = i + 1) {
        if (g.edges[i].from == from && g.edges[i].to == to && !g.edges[i].deleted) {
            g.edges[i].deleted = true
            return
        }
    }
}

// Check if node is valid (exists and not deleted)
isValidNode = (g: &Graph, nodeIdx: i64) -> bool {
    nodeIdx >= 0 && nodeIdx < len(g.nodes) && !g.nodes[nodeIdx].deleted
}

// Check if edge exists (and not deleted)
hasEdge = (g: &Graph, from: i64, to: i64) -> bool {
    for (var i = 0; i < len(g.edges); i = i + 1) {
        if (g.edges[i].from == from && g.edges[i].to == to && !g.edges[i].deleted) {
            return true
        }
    }
    false
}

// Get all outgoing neighbors (skips deleted)
outNeighbors = (g: &Graph, nodeIdx: i64) -> Array<i64> {
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
nodeCount = (g: &Graph) -> i64 {
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
    var children: Array<*TreeNode>
}

Tree = struct {
    var root: *TreeNode
    var parentMap: *Map<i64, i64>>      // child_id -> parent_id
}

buildTree = () -> *Tree {
    val child1 = Heap.new(TreeNode{ 1, 10, [] })
    val child2 = Heap.new(TreeNode{ 2, 20, [] })
    val root = Heap.new(TreeNode{ 0, 0, [child1, child2] })

    var parents = Heap.new(Map{})
    parents.set(1, 0)  // child1's parent is root
    parents.set(2, 0)  // child2's parent is root

    Heap.new(Tree{ root, parents })
}

getParentId = (tree: &Tree, nodeId: i64) -> i64? {
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
    magnitude = (self: &Point) -> i64 {
        sqrt(self.x * self.x + self.y * self.y)
    }

    // Mutable borrow - can modify self, caller keeps ownership
    scale = (self: &&Point, factor: i64) {
        self.x = self.x * factor
        self.y = self.y * factor
    }

    // Takes ownership - self is consumed
    intoArray = (self: *Point) -> Array<i64> {
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

Static methods (no `self` parameter) can return `*Self`:

```slang
Point = class {
    var x: i64
    var y: i64

    // Static factory - no self parameter
    static new = (x: i64, y: i64) -> *Point {
        Heap.new(Point{ x, y })
    }

    // Static factory with default
    static origin = () -> *Point {
        Point.new(0, 0)
    }
}

main = () {
    val p = Point.new(10, 20)         // returns *Point
    val origin = Point.origin()
}
```

## Summary

| Receiver Type | Effect | Caller Ownership |
|---------------|--------|------------------|
| `self: &T` | Immutable borrow | Keeps ownership |
| `self: &&T` | Mutable borrow | Keeps ownership |
| `self: *T` | Takes ownership | Loses access |

## Method Chaining with Consuming Methods

Method chaining works with consuming methods (`self: *T`). The receiver is moved into the method.

```slang
Point = class {
    var x: i64
    var y: i64

    // Consuming method - self is moved in
    intoArray = (self: *Point) -> *Array<i64> {
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
    var next: *Node?       // owns next
    var prev: Weak<Node>?      // weak reference, doesn't own
}

main = () {
    var a = Heap.new(Node{ null, null })
    var b = Heap.new(Node{ null, a.weak() })
    a.next = b

    // Later: b.prev.upgrade() returns *Node? (null if freed)
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
    val maybePoint = Heap.tryNew(Point{ 1, 2 })  // returns *Point?

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
        consume(p)                    // p: *Point
    }

    // points is now empty/invalid
}
```

These would be separate SEPs building on the ownership foundation established here.

# Design Evaluation

After implementing and testing the ownership system with pattern examples (linked lists, binary trees, ownership patterns), here is an assessment of the design.

## Evaluation Summary

| Criterion | Score | Assessment |
|-----------|-------|------------|
| Easy to understand | 7/10 | Clear mental model, some edge cases surprise users |
| Easy to code in | 7/10 | Auto-borrow helps significantly, some restrictions feel limiting |
| Syntax makes sense | 8/10 | Familiar generic syntax, consistent with other languages |
| Flexibility | 7/10 | Good for trees/lists, missing shared ownership patterns |

## What Works Well

### 1. Simple Mental Model
The three pointer types have clear, distinct purposes:
- `*T` = "I own this, it dies when I do"
- `&T` = "I'm borrowing this to read"
- `&&T` = "I'm borrowing this to modify"

### 2. Auto-Borrowing Eliminates Boilerplate
No explicit `.borrow()` or `&` syntax needed:
```slang
printPoint = (p: &Point) { print(p.x) }
val p = Heap.new(Point{ 1, 2 })
printPoint(p)  // Just works - auto-borrows
```

### 3. Ownership Restoration on Reassignment
The pattern `x = fn(x)` naturally works:
```slang
var list: *Node? = null
list = prepend(list, 1)  // list moved to prepend, result assigned back
list = prepend(list, 2)  // works again - ownership was restored
```

### 4. Safe Nullable Traversal
The `?.` operator makes recursive structure traversal clean:
```slang
val v1 = list?.value
val v2 = list?.next?.value
val v3 = list?.next?.next?.value
```

### 5. Automatic Recursive Cleanup
Nested `*T?` fields are automatically freed:
```slang
TreeNode = struct {
    var left: *TreeNode?
    var right: *TreeNode?
    val value: i64
}
// Entire tree freed when root goes out of scope
```

## Potential User Struggles

### 1. Move vs Copy Confusion (Medium)
Users must learn which types are copyable vs move-only:
```slang
val x = 5
val y = x      // OK - i64 is copyable

val p = Heap.new(Point{ 1, 2 })
val q = p      // MOVES p - *Point is move-only
print(p.x)     // ERROR: use of moved value
```

**Mitigation:** Clear error messages explain moves. Consider adding visual move syntax in future (e.g., `val q = move p`).

### 2. Conditional Moves are Conservative (Medium)
Moving in any branch invalidates the variable in all paths:
```slang
val p = Heap.new(Point{ 1, 2 })
if condition {
    consume(p)  // moves p
}
print(p.x)  // ERROR even if condition was false
```

**Rationale:** Simpler to implement and reason about. Flow-sensitive analysis could relax this in future.

### 3. No Moves Inside Loops (Medium)
Cannot move inside loop bodies:
```slang
for i in 0..10 {
    consume(p)  // ERROR: cannot move inside loop
}
```

**Rationale:** Prevents double-free on second iteration. Users must restructure (e.g., use `.copy()` or move before loop).

### 4. References Only in Parameters (Low)
Cannot store `&T` in local variables or struct fields:
```slang
// Cannot do this:
val r: &Point = p  // ERROR

// Must pass directly to functions:
usePoint(p)  // OK - auto-borrows at call site
```

**Rationale:** Prevents dangling references. More complex lifetime analysis could enable this in future.

### 5. Borrow Exclusivity Across Arguments (Low)
Same variable with mixed borrow types fails:
```slang
fn = (a: &&Point, b: &Point) { ... }
fn(p, p)  // ERROR: cannot borrow as both mutable and immutable
```

**Rationale:** Prevents data races. This matches Rust's rules.

## Design Trade-offs

### Simplicity vs Flexibility
We chose **simplicity**: no lifetime annotations, no explicit borrow syntax, references parameter-only. This makes the system easier to learn but less flexible than Rust.

### Safety vs Convenience
We chose **safety**: conservative conditional moves, no loop moves. Users occasionally need to restructure code, but the compiler catches all use-after-move bugs.

### Implicit vs Explicit
We chose **mostly implicit**: auto-borrowing, auto-dereference, implicit moves. This reduces boilerplate but can make moves less visible in code.

## Validated Patterns

The following patterns work well with the current system (see `_programs/patterns/`):

1. **Linked List** - Build with `prepend(list, value)`, traverse with `?.`
2. **Binary Tree** - Recursive `*T?` children, automatic cleanup
3. **Factory Functions** - Return `*T` for caller ownership
4. **Transform Functions** - Take `*T`, return `*T`
5. **Read-Only Access** - Use `&T` parameters
6. **Mutation** - Use `&&T` parameters

## Known Limitations

1. **No Shared Ownership** - Everything is unique or borrowed; no `Rc<T>` yet
2. **No Weak References** - Cannot break reference cycles
3. **No Interior Mutability** - No `Cell<T>` or `RefCell<T>`
4. **No Deep Copy** - `.copy()` not yet implemented

These are documented in Future Work and will be addressed in subsequent SEPs.

# Implementation

## Built-in Types Infrastructure

This SEP introduces several new built-in types (`*T`, `&T`, `Heap`) and built-in methods (`.copy()`). This section documents the infrastructure needed to support these and enable easier addition of built-in types in the future.

### Current Built-in Types

Slang currently has the following built-in types:
- **Primitives:** `i64`, `bool`, `string`, `void`
- **Compound:** `Array<T>`, `T?` (nullable)
- **Functions:** Function types like `(i64, i64) -> i64`

These are currently hardcoded in the type system with special-case handling throughout the compiler.

### New Built-in Types in This SEP

| Type | Category | Description |
|------|----------|-------------|
| `*T` | Generic wrapper | Owned pointer type |
| `&T` | Generic wrapper | Borrowed reference type |
| `Heap` | Singleton type | Built-in allocator with `.new()` method |

### Built-in Type Registry

To enable easier addition of built-in types, introduce a **Built-in Type Registry** in the semantic analyzer:

```go
// compiler/semantic/builtins.go

type BuiltinType struct {
    Name           string
    TypeParams     []string              // e.g., ["T"] for *T
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

// Internal names for pointer types (surface syntax: *T, &T, &&T)
var BuiltinTypes = map[string]BuiltinType{
    "Owned": {  // Surface syntax: *T
        Name:       "Owned",
        TypeParams: []string{"T"},
        Methods: map[string]BuiltinMethod{
            "copy": {
                Name:       "copy",
                Params:     []ParamSpec{},
                ReturnType: TypeSpec{Kind: "Owned", TypeArg: "T"},
                Flags:      MethodFlags{BorrowsReceiver: true},
            },
        },
        Constraints: TypeConstraints{NonNullable: true},
    },
    "Ref": {  // Surface syntax: &T
        Name:       "Ref",
        TypeParams: []string{"T"},
        Methods:    map[string]BuiltinMethod{},
        Constraints: TypeConstraints{NonNullable: true},
    },
    "MutRef": {  // Surface syntax: &&T
        Name:       "MutRef",
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
                ReturnType: TypeSpec{Kind: "Owned", TypeArg: "T"},
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
4. **Constraint validation:** Ensure type arguments satisfy constraints (e.g., `*T?` fails)

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
    "Owned":  &OwnedPointerCodegen{},   // *T
    "Ref":    &RefPointerCodegen{},     // &T
    "MutRef": &MutRefPointerCodegen{},  // &&T
    "Array":  &ArrayCodegen{},
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
| `Weak<T>` | Weak pointer | `.upgrade() -> *T?` |
| `Rc<T>` | Reference counted | `.clone() -> Rc<T>`, `.count() -> i64` |
| `Box<T>` | Simple owned pointer (no custom allocator) | Same as `*T` |
| `Map<K, V>` | Hash map | `.get()`, `.set()`, `.remove()`, `.keys()` |
| `Set<T>` | Hash set | `.add()`, `.remove()`, `.contains()` |
| `Result<T, E>` | Error handling | `.unwrap()`, `.map()`, `.isOk()` |
| `Option<T>` | Explicit optional | `.unwrap()`, `.map()`, `.isSome()` |

### Implementation Priority

1. **Phase 1 (This SEP):** `*T`, `&T`, `Heap` with manual implementation
2. **Phase 2:** Refactor to use registry approach
3. **Phase 3:** Add more built-in types using registry

The registry approach is recommended but not required for initial implementation. The manual approach can be refactored later.

## Step 1: Lexer Changes

Add token support for pointer syntax:
- Recognize `*` as pointer/owned type prefix (context-dependent: multiply vs type prefix)
- Recognize `&` as immutable borrow prefix
- Recognize `&&` as mutable borrow prefix
- Add `Heap` keyword token (or treat as built-in identifier)
- Recognize `.new`, `.copy` as method calls

## Step 2: Parser Changes

Extend the parser to handle pointer expressions and types:
- Parse `*T` and `&T` type syntax
- Parse `Heap.new(expr)` as allocation expression
- Parse `.copy()` method calls
- Parse `var` modifier on function parameters
- Enforce `&T` only in parameter position

## Step 3: Type System Changes

Add pointer types to the semantic analyzer:
- Add `OwnedPointerType` and `RefPointerType` structs
- `Heap.new(expr)` returns `*T` where T is inferred
- Implicit conversion: `*T` → `&T` for function arguments
- Error if `&T` used outside parameter position
- Track `var` modifier on parameters for mutability

## Step 4: Ownership Tracking

Add ownership analysis pass:
- Track variable states: `owned`, `moved`
- `&T` parameters = borrow (caller keeps ownership)
- `*T` parameters = ownership transfer (caller loses access)
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
- `&T` parameters default to immutable
- `&&T` parameters allow mutation
- Borrow exclusivity: one `&&T` OR many `&T`, not both
- No self-referential assignments

## Step 6: Nullable Pointer Integration

Leverage existing nullable type system:
- `*T?` is valid - nullable owned pointer
- `null` assignable to `*T?`
- Safe navigation `?.` works
- Same ownership rules apply

## Step 7: Code Generation

Generate ARM64 assembly:

### Allocation
- Call `_sl_alloc(size)` runtime function
- Store value at allocated address
- Return pointer

### Deallocation
- Call `_sl_free(ptr, size)` at scope exit for owned pointers
- Handle nested structs (free inner pointers first)
- Handle reassignment (free old value before new)

### Copy
- Allocate new memory
- Deep copy contents
- Recursively copy nested pointers

## Memory Allocator

The runtime uses a **bump allocator with size-class free lists** for efficient memory management.

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Arena Chain                              │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐               │
│  │ Arena 3  │───▶│ Arena 2  │───▶│ Arena 1  │───▶ null      │
│  │ (current)│    │          │    │          │               │
│  └──────────┘    └──────────┘    └──────────┘               │
│       │                                                      │
│       ▼                                                      │
│  ┌──────────────────────────────────────────┐               │
│  │ Arena Layout (1MB each):                 │               │
│  │ [next_ptr:8][bump_space:1048560 bytes]   │               │
│  └──────────────────────────────────────────┘               │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                   Size-Class Free Lists                      │
│  Class 0 (16B):  ○──▶○──▶○──▶null                           │
│  Class 1 (32B):  ○──▶null                                   │
│  Class 2 (64B):  ○──▶○──▶○──▶○──▶null                       │
│  Class 3 (128B): null                                        │
│  Class 4 (256B): ○──▶null                                   │
│  Class 5 (512B): null                                        │
│  Class 6 (1KB):  null                                        │
│  Class 7 (2KB):  null                                        │
└─────────────────────────────────────────────────────────────┘
```

### Size Classes

| Class | Size | Use Case |
|-------|------|----------|
| 0 | 16 bytes | Small structs (1-2 fields) |
| 1 | 32 bytes | Small structs (3-4 fields) |
| 2 | 64 bytes | Medium structs, linked list nodes |
| 3 | 128 bytes | Larger structs |
| 4 | 256 bytes | Small arrays |
| 5 | 512 bytes | Medium arrays |
| 6 | 1024 bytes | Larger arrays |
| 7 | 2048 bytes | Large structs |
| 8+ | Aligned | Very large allocations (bump only) |

### Allocation Algorithm

```
_sl_alloc(size):
  1. Get size class and rounded size
  2. If size class < 8 (not large):
     a. Check free list for this class
     b. If non-empty: pop head, return it (O(1))
  3. Bump allocate:
     a. If bump_ptr + size > arena_end:
        - Allocate new 1MB arena via mmap
        - Chain to previous arena
     b. result = bump_ptr
     c. bump_ptr += size
     d. Return result
```

### Deallocation Algorithm

```
_sl_free(ptr, size):
  1. Get size class for size
  2. If size class >= 8: return (large allocs not recycled)
  3. Push ptr onto free list head (O(1))
     - ptr.next = free_list[class]
     - free_list[class] = ptr
```

### Arena Retention Policy

**Arenas are allocated but never deallocated.** This is intentional:

| Approach | Pros | Cons |
|----------|------|------|
| **Retain arenas (current)** | Simple, fast, no bookkeeping | Memory stays at high-water mark |
| Deallocate empty arenas | Returns memory to OS | Complex ref counting, slower |

**Rationale:**
- Short-lived programs (compilers, scripts): doesn't matter
- Servers with steady-state allocation: arenas reach working set and stay there
- Extreme memory spikes: could add explicit `Heap.shrink()` API later
- Common pattern: jemalloc, tcmalloc also retain memory pools

### Performance Characteristics

| Operation | Complexity | Notes |
|-----------|------------|-------|
| Allocation (free list hit) | O(1) | Pop from free list |
| Allocation (bump) | O(1) | Increment pointer |
| Allocation (new arena) | O(1) | mmap syscall |
| Deallocation | O(1) | Push to free list |

### Memory Efficiency

| Scenario | Old (mmap each) | New (bump allocator) |
|----------|-----------------|----------------------|
| 1,000 small allocs | ~16 MB | ~1 MB |
| 50,000 small allocs | ~800 MB | ~4 MB |
| 5M alloc/free cycles | ~800 MB peak | ~1 MB flat |

The bump allocator reduces memory usage by **~200x** for typical allocation patterns.

### Runtime Functions

| Function | Purpose |
|----------|---------|
| `_sl_heap_init` | Initialize first arena (called at `_start`) |
| `_sl_alloc(size)` | Allocate memory, returns pointer in x0 |
| `_sl_free(ptr, size)` | Return memory to free list |
| `_sl_get_size_class(size)` | Map size to class index |
| `_sl_arena_grow` | Allocate new 1MB arena |

## Error Handling

```slang
val p = Heap.new(42)
print(p.x)                                // Error: *i64 has no fields

val p = Heap.new(Point{ 1, 2 })
print(p[0])                               // Error: *Point not indexable

val maybeP: *Point? = null
print(maybeP.x)                           // Error: use ?.x for nullable pointer

var q = Heap.new(Point{ 1, 2 })
q = q                                     // Error: cannot assign variable to itself

foo(p, p)                                 // Error: 'p' moved in first argument
```

## Quick Reference

| Expression | Type of `p` | Result Type | Notes |
|------------|-------------|-------------|-------|
| `p.x` | `*Struct` or `&Struct` | field type | Auto-deref |
| `p[i]` | `*Array<T>` where T is primitive | `T` | Copy of element |
| `p[i]` | `*Array<*T>` | `&T` | Borrows element |
| `p.field` | where field is `*T` | `&T` | Auto-borrow through ref |
| `p?.x` | `*Struct?` | field type? | Safe navigation |
| `p?[i]` | `*Array<T>?` | `T?` | Safe navigation |
| `p == null` | `*T?` | `bool` | Null check |
| `p == q` | `*T` | `bool` | Identity (address) comparison |

# Alternatives

1. **Generic wrapper types (`Own<T>`, `Ref<T>`, `MutRef<T>`)**: Originally considered using generic wrapper types like `Own<Point>`, `Ref<Point>`, `MutRef<Point>`. Rejected due to verbosity with nested types (e.g., `Own<Array<Own<Point>>>`). The symbol-based syntax (`*T`, `&T`, `&&T`) is more concise.

2. **Rust-style references (`&T`, `&mut T`)**: More complex borrow checker required. Rejected for MVP simplicity. We use `&T` for immutable borrows but `&&T` for mutable borrows (instead of `&mut T`).

3. **`&var T` or `&mut T` for mutable borrows**: Originally considered using `&var T` or `&mut T` to indicate mutable borrows. Rejected in favor of `&&T` for conciseness - "double ampersand = more power" is an intuitive mnemonic.

4. **Implicit dereference everywhere**: Would hide pointer semantics too much. Auto-deref for field access provides convenience where it's unambiguous.

5. **Manual memory management from start**: Would complicate MVP. Starting with allocation-only allows proving the design before adding deallocation complexity.

6. **`ptr::new(value)` syntax**: Simpler but doesn't allow for custom allocators. The `Heap.new(value)` design anticipates the allocator interface pattern.

## Syntax Choice Rationale

The final syntax uses symbols for pointer types:

| Type | Symbol | Meaning |
|------|--------|---------|
| Owned pointer | `*T` | You control the lifetime |
| Immutable borrow | `&T` | Read-only borrowed reference |
| Mutable borrow | `&&T` | Can mutate `var` fields |

**Why symbols over generics?**
- **Conciseness**: `*Array<*Point>` vs `Own<Array<Own<Point>>>`
- **Familiarity**: `*` and `&` are well-known from C/Rust
- **No keyword mixing**: `&&T` is cleaner than `&var T`
- **Visual hierarchy**: `&` < `&&` suggests "more ampersands = more access"

**Why `&&T` for mutable borrow?**
- Concise (2 characters vs `&mut T`)
- Symmetric with `&T`
- In type position, `&&` doesn't conflict with logical AND (expression context only)

# Future Work: Allocator Interface

The `Heap` type is designed to be the first implementation of an allocator interface. Future work could include:

```slang
// Future: Allocator interface (requires interface/trait system)
Allocator = interface {
    new<T>(value: T) -> *T
    alloc<T>() -> *T              // uninitialized allocation
    free<T>(p: *T)                // explicit deallocation
}

// Built-in implementations
Heap: Allocator        // System heap (mmap/brk)
Arena: Allocator       // Arena/bump allocator
Pool<T>: Allocator     // Fixed-size pool allocator

// Usage with custom allocator
createWithAllocator = (alloc: Allocator, x: i64, y: i64) -> *Point {
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
- Token recognition for `*`, `&`, `&&`, `Heap`
- `*T`, `&T`, and `&&T` type parsing
- `Heap.new(expr)` expressions
- `.copy()` method calls
- `var` modifier on function parameters

## Semantic Tests
- Type inference for pointer types
- `Heap` type checking
- Auto-dereference rules
- Auto-borrowing from `*T` to `&T`
- Nullable pointer rules
- `&T` only allowed in parameter position
- `var` enables mutation on parameters
- Pointer comparison (`==`) is identity (address) comparison
- Nullable pointer comparison to `null`
- Array indexing returns copy for primitives, borrow for `*T`
- `*T?` is invalid (T must be non-nullable)
- Safe navigation `?.` on non-nullable is error
- Array of nullable pointers: indexing returns nullable borrow
- Array literal type inference with null elements

## Ownership Tests
- Move on assignment (`val q = p` invalidates `p`)
- Use-after-move detection
- `&T` parameters borrow (caller keeps ownership)
- `*T` parameters take ownership (caller loses access)
- Cannot mutate through immutable `&T`
- `&&T` allows mutation
- `val` binding can create mutable borrows (`val` only controls reassignment)
- Borrow exclusivity (one `&&T` OR many `&T`, not both)
- Mixed borrow + move in same call is error
- Nested ownership (struct containing `*T` fields)
- Field access through `&T` auto-borrows `*T` fields
- Chained field access propagates mutability
- Closures capturing `*T` move ownership into closure
- Closures cannot capture `&T` (error)
- Generics with `&T` in field position (error)
- Temporary lifetimes extend to end of statement
- Loop iteration borrows container
- Mutable iteration produces `&&T` for `var` arrays
- Cannot reassign while borrowed (including during iteration)
- Self-assignment is error
- Overlapping moves in assignment is error
- Left-to-right evaluation order with move tracking (borrows are fine)
- Early return drops owned values in reverse order
- Nullable pointer smart cast inside null check
- Smart cast in while loop condition and body
- Nested null checks with auto-borrow
- Operators auto-dereference `*primitive`
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
- Ownership transfer with `*T` parameters
- Auto-borrowing with `&T` and `&&T` parameters
- Memory freed correctly (no leaks in simple cases)
- Pointer identity comparison (`p == q`)
- Closures capturing owned pointers
- Loop iteration over pointer arrays
- Temporary lifetime in chained expressions
- Method calls with different receiver types (SEP 7)
- Method chaining with consuming methods
- Operators auto-dereference *primitive
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
  - `*T?` type declaration
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

    // Auto-borrow when passing to &T parameter
    printRef = (r: &i64) { /* can read r */ }
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
    val points: Array<*Point> = [
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
    var next: *Node?                  // nullable, owned by this node
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

Shows a function that allocates and returns a pointer. Returns `*T` to transfer ownership.

```slang
Point = struct {
    val x: i64
    val y: i64
}

createPoint = (x: i64, y: i64) -> *Point {
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
    var spouse: *Person?              // may or may not have a spouse
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

## Example 8: Borrowing with &T and Ownership with *T

Shows the difference between borrowing (`&T`) and ownership transfer (`*T`).

```slang
Point = struct {
    var x: i64
    var y: i64
}

// Immutable borrow - cannot modify
printPoint = (p: &Point) {
    print(p.x)
    print(p.y)
    // p.x = 100                          // Error: p is not var
}

// Mutable borrow - can modify, caller keeps ownership
scalePoint = (p: &&Point, factor: i64) {
    p.x = p.x * factor
    p.y = p.y * factor
}

// Takes ownership - caller loses access
consume = (p: *Point) {
    print(p.x)
}                                         // p freed here

main = () {
    var p = Heap.new(Point{ 10, 20 })

    printPoint(p)                         // borrows as &Point
    printPoint(p)                         // can borrow multiple times

    scalePoint(p, 2)                      // borrows as &&Point
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
    var left: *TreeNode?
    var right: *TreeNode?
}

// Creates a new leaf node
leaf = (value: i64) -> *TreeNode {
    Heap.new(TreeNode{ value, null, null })
}

// Creates a new internal node - children move into the node
node = (value: i64, left: *TreeNode, right: *TreeNode) -> *TreeNode {
    Heap.new(TreeNode{ value, left, right })  // left and right move in
}

// Borrows tree to compute sum (&T = immutable borrow)
sum = (t: &TreeNode) -> i64 {
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
    // val binding - can mutate var fields (val only controls reassignment)
    val p = Heap.new(Point{ 1, 2 })
    // p.x = 10                           // Error: x is val field
    p.y = 20                              // OK: y is var field
    // p = other                          // Error: cannot reassign val binding

    // var binding - can also reassign the binding itself
    var q = Heap.new(Point{ 1, 2 })
    // q.x = 10                           // Error: x is val field
    q.y = 30                              // OK: y is var field
    q = Heap.new(Point{ 5, 6 })           // OK: q is var, can reassign

    print(q.y)                            // prints: 6
}
```

## Example 11: Method-Style Ownership Transfer

Shows using `*T` parameters to transfer ownership into struct fields.

```slang
Node = struct {
    val value: i64
    var next: *Node?
}

// Takes ownership of 'next' and stores it in the node
setNext = (node: &&Node, next: *Node) {
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
    val n: *Point? = null
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
        print(p.x)                         // p: &Point
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

createPoint = (x: i64, y: i64) -> *Point {
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
        val inner: *Point
    }

    createOuter = () -> *Outer {
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
    val points: *Array<*Point> = Heap.new([
        Heap.new(Point{ 1, 2 }),
        Heap.new(Point{ 3, 4 }),
        Heap.new(Point{ 5, 6 })
    ])

    val p = points[0]                      // p: &Point (borrow)
    print(p.x)                             // prints: 1

    val q = points[0]                      // q: &Point (can borrow again)
    print(q.x)                             // prints: 1 (same element)

    // Array still owns all elements
    print(points[0].x)                     // prints: 1

    // To replace an element, use .set()
    val old = points.set(0, Heap.new(Point{ 100, 200 }))
    // old: *Point (the replaced element)
    print(points[0].x)                     // prints: 100

    // To remove an element, use .remove()
    val removed = points.remove(0)         // removed: *Point, shifts elements
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
    var data: *Point
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

mixedBorrow = (a: &&Point, b: &Point) {
    a.x = b.x + 1
}

// ❌ Error: mixed borrow and move
bad6 = () {
    val p = Heap.new(Point{ 1, 2 })
    borrowAndMove(p, p)                    // Error: cannot borrow and move same value
}

borrowAndMove = (r: &Point, p: *Point) {
    print(r.x)
}

// ✅ OK: mutable borrow from val binding (val only controls reassignment)
good_mutate = () {
    val p = Heap.new(Point{ 1, 2 })        // val binding
    mutate(p)                              // OK: val only prevents reassigning p
    print(p.x)                             // prints 10
}

mutate = (p: &&Point) {
    p.x = 10
}

// ❌ Error: reassign while borrowed
bad8 = () {
    var arr = Heap.new([1, 2, 3])
    for x in arr {
        arr = Heap.new([4, 5, 6])          // Error: cannot reassign while borrowed
    }
}

// ✅ OK: operators auto-dereference *primitive
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

readBoth = (a: &Point, b: &Point) {
    print(a.x + b.x)
}

// ✅ OK: nullable smart cast
good2 = () {
    val p: *Point? = maybeGet()
    if (p != null) {
        print(p.x)                         // OK: smart cast to *Point
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
    val p: *Point? = maybeGetPoint()

    // Without null check - must use ?.
    print(p?.x)                            // OK: safe navigation

    // With null check - smart cast to non-null
    if (p != null) {
        print(p.x)                         // OK: p is *Point here
        print(p.y)                         // OK: no ?. needed

        // Ownership still applies
        val q = p                          // moves p
        // print(p.x)                      // Error: p was moved
    }

    // After if: conditional move applies
    // print(p?.x)                         // Error: p may have been moved

    // Reassignment vs move
    var r: *Point? = maybeGetPoint()
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
    val inner: *Inner
}

Container = struct {
    val outer: *Outer
}

// Immutable access - all borrows are immutable
readDeep = (c: &Container) {
    // c is &Container
    // c.outer is &Outer (auto-borrow of *Outer field)
    // c.outer.inner is &Inner (auto-borrow of *Inner field)
    val v = c.outer.inner.value            // v: i64 (copy of primitive)
    print(v)
}

// Mutable access - mutability propagates through chain
mutateDeep = (c: &&Container) {
    // c is &&Container
    // c.outer is &&Outer (inherits mutability)
    // c.outer.inner is &&Inner (inherits mutability)
    c.outer.inner.value = 100              // OK: can mutate through &&T chain
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
