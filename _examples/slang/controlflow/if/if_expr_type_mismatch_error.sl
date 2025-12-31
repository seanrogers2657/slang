// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=different types
// Tests that if expression with mismatched types produces error
main = () {
    val x = if true { 42 } else { true }
    exit(0)
}
