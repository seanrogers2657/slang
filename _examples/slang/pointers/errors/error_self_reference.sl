// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot bind owned value
// Test: Cannot self-assign an owned pointer (it would alias its own owner)
Point = struct {
    val x: s64
    val y: s64
}

main = () {
    var p = new Point{ 10, 20 }
    p = p  // Error: self-assignment of an owned value (single owner)
}
