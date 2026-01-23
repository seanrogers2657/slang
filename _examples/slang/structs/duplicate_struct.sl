// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=struct 'Point' is already declared
Point = struct {
    val x: s64
}

Point = struct {
    val y: s64
}

main = () {}
