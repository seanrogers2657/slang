// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot assign
// Test: Cannot self-assign an owned pointer
Point = struct {
    val x: s64
    val y: s64
}

main = () {
    var p = new Point{ 10, 20 }
    p = p  // Error: self-assignment of move-only type
}
