// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=out of range for i16
fn main(): void {
    val x: i16 = 32768
    print(x)
}
