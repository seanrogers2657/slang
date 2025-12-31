// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=requires boolean operands
// Tests that logical operators on integers are rejected
main = () {
    val x = 5 && 3
}
