// @test: exit_code=0
// @test: stdout=10\n
// Test: Nested call with value return is OK
// The inner & borrow ends before the outer && borrow starts
Point = struct {
    var x: s64
}

getX = (p: &Point) -> s64 {
    return p.x
}

setX = (p: &&Point, v: s64) {
    p.x = v
}

main = () {
    val p = new Point{ 10 }

    // Nested: getX(p) returns i64 value, borrow ends
    // Then setX gets fresh && borrow
    setX(p, getX(p))  // OK: sequential, not simultaneous

    print(p.x)  // 10
}
