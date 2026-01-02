// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot infer type from null
// Test that bare null without type annotation is an error
main = () {
    val x = null
}
