// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=struct 'Point' is already declared
Point = struct {
    val x: i64
}

Point = struct {
    val y: i64
}

main = () {}
