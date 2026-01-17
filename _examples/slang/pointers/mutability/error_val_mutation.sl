// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot assign to immutable field
// Test: Cannot mutate VAL FIELD (immutable field), even through mutable binding
Point = struct {
    val x: i64  // val field cannot be mutated
    var y: i64
}

main = () {
    var p = Heap.new(Point{ 10, 20 })
    p.x = 100  // Error: cannot assign to immutable field 'x'
}
