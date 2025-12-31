// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=out of range for u8
main = () {
    val x: u8 = 256
    print(x)
}
