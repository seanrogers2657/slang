// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=not exhaustive
fn main(): void {
    val x = 5
    when {
        x > 10 -> exit(100)
        x > 5 -> exit(50)
    }
}
