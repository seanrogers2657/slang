// @test: expect_error=true
// @test: error_stage=parser
// @test: error_contains=cannot mix positional and named arguments
struct Point(val x: i64, val y: i64)

fn main(): void {
    val p = Point(10, y: 20)
}
