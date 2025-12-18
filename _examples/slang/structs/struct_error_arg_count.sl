// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=has 2 field(s), but 1 argument(s) were provided
struct Point(val x: i64, val y: i64)

fn main(): void {
    val p = Point(1)
}
