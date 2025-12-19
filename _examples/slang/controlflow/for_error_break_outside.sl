// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains='break' statement not inside a loop
fn main(): void {
    break
}
