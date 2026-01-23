// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot assign null to non-nullable
// Test that null cannot be assigned to non-nullable type
main = () {
    val x: s64 = null
}
