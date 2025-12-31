// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=requires numeric operands
// Tests that arithmetic on booleans is rejected
main = () {
    val x = true + false
}
