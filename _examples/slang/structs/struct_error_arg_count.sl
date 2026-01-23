// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=has 2 field(s), but 1 argument(s) were provided
Point = struct {
    val x: s64
    val y: s64
}

main = () {
    val p = Point{ 1 }
}
