// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot assign to element of immutable array
fn main(): void {
    val arr = [1, 2, 3]
    arr[0] = 100
}
