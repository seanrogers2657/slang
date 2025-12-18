// @test: expect_error=true
// @test: error_stage=lexer
// @test: error_contains=bitwise & not supported
fn main(): void {
    val x = 5 & 3
}
