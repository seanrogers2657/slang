// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot index non-array type
main = () {
    val x = 5
    print(x[0])
}
