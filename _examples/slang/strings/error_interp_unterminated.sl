// @test: expect_error=true
// @test: error_stage=lexer
// @test: error_contains=unterminated string literal
// An interpolation that is never closed is a lexer error.
main = () {
    print("hi ${name")
}
