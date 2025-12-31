// @test: expect_error=true
// @test: error_stage=parser
// @test: error_contains=cannot mix positional and named arguments
Point = struct {
    val x: i64
    val y: i64
}

main = () {
    val p = Point{ 10, y: 20 }
}
