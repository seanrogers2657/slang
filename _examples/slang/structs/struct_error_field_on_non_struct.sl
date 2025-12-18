// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot access field 'value' on non-struct type
fn main(): void {
    val x = 42
    print(x.value)
}
