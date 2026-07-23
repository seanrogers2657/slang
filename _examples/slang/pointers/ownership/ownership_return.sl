// @test: exit_code=0
// @test: stdout=100\n200\n
// Test: a factory returns a Point by value — the result is copied to the caller
// (owned heap cannot be returned).
Point = struct {
    val x: s64
    val y: s64
}

createPoint = (x: s64, y: s64) -> Point {
    return Point{ x, y }
}

main = () {
    val p = createPoint(100, 200)
    print(p.x)
    print(p.y)
}
