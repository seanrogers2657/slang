// @test: expect_error=true
// @test: error_stage=parser
// @test: error_contains=expected ')' to close grouping expression
fn main(): void {
    val x = (5 + 3
}
