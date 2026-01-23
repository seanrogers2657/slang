// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=out of range for s8
main = () {
    val x: s8 = 200
    print(x)
}
