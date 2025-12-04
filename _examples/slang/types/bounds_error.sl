// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=out of range for i8
fn main(): void {
    val x: i8 = 200
    print x
}
