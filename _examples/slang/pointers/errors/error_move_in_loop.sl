// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot move
// Test: Moving inside a loop should produce an error
Point = struct {
    val x: i64
    val y: i64
}

consumePoint = (p: *Point) -> i64 {
    return p.x + p.y
}

main = () {
    val p = Heap.new(Point{ 10, 20 })
    for var i = 0; i < 3; i = i + 1 {
        consumePoint(p)  // Error: cannot move inside loop
    }
}
