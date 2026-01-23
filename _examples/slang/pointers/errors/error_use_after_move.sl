// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=moved value
// Test: Use-after-move should produce an error
Point = struct {
    val x: s64
    val y: s64
}

main = () {
    val p = Heap.new(Point{ 10, 20 })
    val q = p  // p is moved to q
    print(p.x)  // Error: p was moved
}
