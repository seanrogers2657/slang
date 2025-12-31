// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=out of bounds
main = () {
    val arr = [1, 2, 3]
    print(arr[5])
}
