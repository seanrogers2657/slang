// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot assign boolean to variable of type s64
Point = struct {
    val x: s64
    val y: s64
}

main = () {
    val p = Point{ true, 2 }
}
