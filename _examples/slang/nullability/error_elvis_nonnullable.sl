// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=requires nullable type
// Error: left operand must be nullable
main = () {
    val x: s64 = 10
    val result = x ?: 42
    print(result)
}
