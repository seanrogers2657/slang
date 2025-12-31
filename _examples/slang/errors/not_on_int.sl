// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=requires boolean operand
// Tests that logical NOT on integers is rejected
main = () {
    val x = !5
}
