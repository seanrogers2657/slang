// Ownership Patterns
// Demonstrates: choosing &T (borrow to read) vs &&T (borrow to mutate), and
// returning new data by value. Under the scope-frees-it model ownership never
// transfers: functions borrow their arguments and return values (copied to the
// caller); an owned `new` local is freed at the end of its scope, and .copy()
// makes an independent owned duplicate.

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

// Multiple &T params - can borrow the same value more than once
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
// PATTERN 3: Borrow to read and derive a value
// There is no "take ownership" — consuming/move parameters don't exist. A
// function that just needs to read data borrows it with &T and returns a
// scalar; the caller keeps the value.
// =============================================================================

field_sum = (p: &Point) -> s64 {
    return p.x + p.y
}

// =============================================================================
// PATTERN 4: Return a new value (-> T)
// Use when: Function produces new data. Owned heap can't be returned, so a
// factory (or a derive like clone/midpoint) returns the struct by value; the
// result is copied to the caller.
// =============================================================================

create_point = (x: s64, y: s64) -> Point {
    return Point{ x, y }
}

clone = (p: &Point) -> Point {
    return Point{ p.x, p.y }
}

midpoint = (a: &Point, b: &Point) -> Point {
    return Point{
        (a.x + b.x) / 2,
        (a.y + b.y) / 2
    }
}

// =============================================================================
// PATTERN 5: Transform borrowed input into a new value (&T -> T)
// Use when: Function derives new data from a borrowed input. No ownership
// passes through — the input is borrowed, the output is returned by value.
// =============================================================================

doubled = (p: &Point) -> Point {
    val result = Point{ p.x * 2, p.y * 2 }
    return result
}

// =============================================================================
// MAIN: Demonstrate patterns
// =============================================================================

main = () {
    print("=== Pattern 1: Borrow to read ===")
    // An owned `new` local auto-borrows into the &T/&&T functions below.
    val p1 = new Point{ 3, 4 }
    assert(distance(p1) == 25, "distance should be 25 (3*3 + 4*4)")
    assert(are_same_point(p1, p1), "point should equal itself")
    print(distance(p1))            // 25
    print(are_same_point(p1, p1))  // true

    print("=== Pattern 2: Borrow to mutate ===")
    var p2 = new Point{ 10, 20 }
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

    print("=== Pattern 3: Borrow to read and derive a value ===")
    val p3 = new Point{ 7, 8 }
    val sum = field_sum(p3)
    assert(sum == 15, "sum should be 15 (7+8)")
    print(sum)  // 15
    // p3 still valid — field_sum only borrowed it

    print("=== Pattern 4: Return a new value ===")
    // A factory returns a Point by value; read its fields directly.
    val made = create_point(100, 200)
    assert(made.x == 100, "made.x should be 100")
    print(made.x)  // 100
    // clone borrows an owned point and returns an independent value copy.
    val cl = clone(p1)
    assert(cl.x == 3, "clone of p1 should have x = 3")
    print(cl.x)  // 3

    print("=== Pattern 5: Independent copy, then transform ===")
    // .copy() produces an independent owned *Point — the way to duplicate owned
    // data when moves don't exist.
    val src = new Point{ 100, 200 }
    val dup = src.copy()
    scale(src, 0)
    assert(src.x == 0, "src should be scaled to 0")
    assert(dup.x == 100, "dup should be unaffected by src")
    print(src.x)  // 0
    print(dup.x)  // 100 (independent copy)

    // Borrowing functions that return new values: midpoint and doubled.
    val mid = midpoint(src, dup)   // ((0+100)/2, (0+200)/2) = (50, 100)
    assert(mid.x == 50, "midpoint x should be 50")
    print(mid.x)  // 50

    val dbl = doubled(dup)         // (200, 400)
    assert(dbl.x == 200, "doubled x should be 200")
    assert(dbl.y == 400, "doubled y should be 400")
    print(dbl.x)  // 200
    print(dbl.y)  // 400

    print("Ownership patterns test passed!")
}
