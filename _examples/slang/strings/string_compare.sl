// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=requires numeric operands
main = () {
    "hello" == "hello"
}
