// @test: expect_error=true
// @test: error_stage=parser
// @test: error_contains=expected newline or '}' after statement
// Note: Single & is now a valid token for type syntax (&T), but not valid as an operator
main = () {
    val x = 5 & 3
}
