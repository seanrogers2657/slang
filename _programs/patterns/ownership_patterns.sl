// Ownership Patterns
// Demonstrates: When to use * vs & vs && in function signatures

Point = struct {
    var x: s64
    var y: s64
}

// =============================================================================
// PATTERN 1: Borrow to read (&T)
// Use when: Function only needs to read data, caller keeps ownership
// =============================================================================

distance = (p: &Point) -> s64 {
    return p.x * p.x + p.y * p.y
}

// Multiple Ref params - can borrow same value multiple times
are_same_point = (a: &Point, b: &Point) -> bool {
    return a.x == b.x && a.y == b.y
}

// =============================================================================
// PATTERN 2: Borrow to mutate (&&T)
// Use when: Function needs to modify data, caller keeps ownership
// =============================================================================

scale = (p: &&Point, factor: s64) {
    p.x = p.x * factor
    p.y = p.y * factor
}

translate = (p: &&Point, dx: s64, dy: s64) {
    p.x = p.x + dx
    p.y = p.y + dy
}

reset = (p: &&Point) {
    p.x = 0
    p.y = 0
}

// =============================================================================
// PATTERN 3: Take ownership (*T)
// Use when: Function consumes the value or stores it somewhere
// =============================================================================

consume = (p: *Point) -> s64 {
    // p is freed when this function returns
    return p.x + p.y
}

// =============================================================================
// PATTERN 4: Return ownership (-> *T)
// Use when: Function creates new data for caller to own
// =============================================================================

create_point = (x: s64, y: s64) -> *Point {
    return Heap.new(Point{ x, y })
}

clone = (p: &Point) -> *Point {
    return Heap.new(Point{ p.x, p.y })
}

midpoint = (a: &Point, b: &Point) -> *Point {
    return Heap.new(Point{
        (a.x + b.x) / 2,
        (a.y + b.y) / 2
    })
}

// =============================================================================
// PATTERN 5: Transform and return (*T -> *T)
// Use when: Function transforms data, ownership passes through
// =============================================================================

doubled = (p: *Point) -> *Point {
    val result = Heap.new(Point{ p.x * 2, p.y * 2 })
    // p is freed here
    return result
}

// =============================================================================
// MAIN: Demonstrate patterns
// =============================================================================

main = () {
    print("=== Pattern 1: Borrow to read ===")
    val p1 = create_point(3, 4)
    assert(distance(p1) == 25, "distance should be 25 (3*3 + 4*4)")
    assert(are_same_point(p1, p1), "point should equal itself")
    print(distance(p1))          // 25 (3*3 + 4*4)
    print(are_same_point(p1, p1))  // true
    // p1 still valid

    print("=== Pattern 2: Borrow to mutate ===")
    var p2 = create_point(10, 20)
    scale(p2, 2)
    assert(p2.x == 20, "x should be 20 after scale")
    assert(p2.y == 40, "y should be 40 after scale")
    print(p2.x)  // 20
    print(p2.y)  // 40
    translate(p2, 5, 5)
    assert(p2.x == 25, "x should be 25 after translate")
    assert(p2.y == 45, "y should be 45 after translate")
    print(p2.x)  // 25
    print(p2.y)  // 45
    // p2 still valid

    print("=== Pattern 3: Take ownership ===")
    val p3 = create_point(7, 8)
    val sum = consume(p3)
    assert(sum == 15, "sum should be 15 (7+8)")
    print(sum)  // 15
    // p3 is now invalid

    print("=== Pattern 4: Return ownership ===")
    val p4 = create_point(100, 200)
    val p5 = clone(p4)
    assert(p5.x == 100, "cloned x should be 100")
    print(p5.x)  // 100
    scale(p4, 0)
    assert(p4.x == 0, "scaled x should be 0")
    assert(p5.x == 100, "clone should be unaffected")
    print(p4.x)  // 0
    print(p5.x)  // 100 (clone unaffected)

    val mid = midpoint(p4, p5)
    assert(mid.x == 50, "midpoint x should be 50")
    print(mid.x)  // 50

    print("=== Pattern 5: Transform and return ===")
    val p6 = create_point(5, 10)
    val p7 = doubled(p6)
    // p6 is now invalid
    assert(p7.x == 10, "doubled x should be 10")
    assert(p7.y == 20, "doubled y should be 20")
    print(p7.x)  // 10
    print(p7.y)  // 20

    print("Done")
    print("Ownership patterns test passed!")
}
