// @test: exit_code=0
// @test: stdout=100\n200\n
// Test: Returning *T from a function transfers ownership to caller
Point = struct {
    val x: s64
    val y: s64
}

createPoint = (x: s64, y: s64) -> *Point {
    return new Point{ x, y }
}

main = () {
    val p = createPoint(100, 200)
    print(p.x)
    print(p.y)
}
