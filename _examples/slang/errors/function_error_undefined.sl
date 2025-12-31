// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=undefined function
main = () {
    val x = unknown_function()
}
