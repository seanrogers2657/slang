// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=out of range for s16
main = () {
    val x: s16 = 32768
    print(x)
}
