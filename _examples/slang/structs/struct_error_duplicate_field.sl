// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=field 'x' specified multiple times
Point = struct {
    val x: i64
    val y: i64
}

main = () {
    val p = Point{ x: 10, x: 20 }
}
