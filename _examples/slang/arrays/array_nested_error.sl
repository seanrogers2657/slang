// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=nested arrays are not supported
main = () {
    val arr = [[1, 2], [3, 4]]
    print(0)
}
