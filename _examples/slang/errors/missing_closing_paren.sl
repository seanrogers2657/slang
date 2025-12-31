// @test: expect_error=true
// @test: error_stage=parser
// @test: error_contains=expected ')' to close grouping expression
main = () {
    val x = (5 + 3
}
