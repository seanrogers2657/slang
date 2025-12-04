// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=return type mismatch
fn get_number(): int {
    return "hello"
}

fn main(): void {
    val x = get_number()
}
