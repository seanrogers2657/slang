// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=must have an else branch
// Tests that if expression without else produces error
fn main(): void {
    val x = if true { 42 }
    exit(x)
}
