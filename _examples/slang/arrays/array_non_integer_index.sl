// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=array index must be integer
fn main(): void {
    val arr = [1, 2, 3]
    print(arr[true])
}
