// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=integer literal -1 out of range for u8
main = () {
    val x = -1u8
    print(x)
}
