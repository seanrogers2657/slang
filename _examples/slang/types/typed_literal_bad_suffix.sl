// @test: expect_error=true
// @test: error_stage=lexer
// @test: error_contains=invalid integer literal suffix 'u7'
main = () {
    print(5u7)
}
