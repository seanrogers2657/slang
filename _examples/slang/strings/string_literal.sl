// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=requires numeric operands
fn main(): void {
    "hello" + "world"
}
