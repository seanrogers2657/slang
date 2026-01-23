// @test: expect_error=true
// @test: error_stage=parser
// @test: error_contains=cannot mix positional and named arguments
Point = struct {
    val x: s64
    val y: s64
}

main = () {
    val p = Point{ 10, y: 20 }
}
