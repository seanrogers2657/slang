// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=out of range for i8
main = () {
    val x: i8 = 200
    print(x)
}
