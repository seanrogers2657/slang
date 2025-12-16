// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=maximum allowed is 8
fn tooMany(a: i64, b: i64, c: i64, d: i64, e: i64, f: i64, g: i64, h: i64, i: i64): void {
    print(a)
}

fn main(): void {
    tooMany(1, 2, 3, 4, 5, 6, 7, 8, 9)
}
