// @test: exit_code=0
// @test: stdout=100\n200\n
// Test: Returning *T from a function transfers ownership to caller
Point = struct {
    val x: i64
    val y: i64
}

createPoint = (x: i64, y: i64) -> *Point {
    return Heap.new(Point{ x, y })
}

main = () {
    val p = createPoint(100, 200)
    print(p.x)
    print(p.y)
}
