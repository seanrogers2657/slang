// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot index non-array type
fn main(): void {
    val x = 5
    print(x[0])
}
