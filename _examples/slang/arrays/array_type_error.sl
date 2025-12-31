// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=array element type mismatch
main = () {
    val arr = [1, "hello"]
}
