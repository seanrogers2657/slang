// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=expects 2 arguments
fn add(a: int, b: int): int {
    return a + b
}

fn main(): void {
    val x = add(1)
}
