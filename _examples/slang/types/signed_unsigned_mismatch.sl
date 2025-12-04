// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=requires operands of the same type
fn main(): void {
    val a: i32 = 10
    val b: u32 = 20
    val c = a + b
}
