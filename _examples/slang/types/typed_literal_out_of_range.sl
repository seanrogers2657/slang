// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=integer literal 256 out of range for u8
main = () {
    print(256u8)
}
