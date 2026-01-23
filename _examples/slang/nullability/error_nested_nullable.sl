// @test: expect_error=true
// @test: error_stage=parser
// @test: error_contains=nested nullable types are not allowed
// Test that nested nullable types are not allowed
main = () {
    val x: s64?? = null
}
