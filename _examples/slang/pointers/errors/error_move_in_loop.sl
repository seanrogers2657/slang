// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=owned pointers (*T) cannot be parameters
// Test: An owned pointer (*T) cannot be a function parameter.
Point = struct {
    val x: s64
    val y: s64
}

consumePoint = (p: *Point) -> s64 {  // Error: *T cannot be a parameter
    return p.x + p.y
}

main = () {
    val p = new Point{ 10, 20 }
    for var i = 0; i < 3; i = i + 1 {
        consumePoint(p)
    }
}
