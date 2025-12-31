// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=out of range for i64
fn main(): void {
    // Untyped integer literal exceeds i64 max (9223372036854775807)
    val x = 9223372036854775808
    print(x)
}
