// @test: expect_error=true
// @test: error_stage=lexer
// @test: error_contains=invalid suffix 'u8' on float literal
main = () {
    print(1.5u8)
}
