// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot assign to immutable variable
main = () {
    val x = 5
    x = 10
}
