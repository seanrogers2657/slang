// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=nested arrays are not supported
fn main(): void {
    val arr = [[1, 2], [3, 4]]
    print(0)
}
