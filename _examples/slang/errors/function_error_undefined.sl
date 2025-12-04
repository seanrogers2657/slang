// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=undefined function
fn main(): void {
    val x = unknown_function()
}
