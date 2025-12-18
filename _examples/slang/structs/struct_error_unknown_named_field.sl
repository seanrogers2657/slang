// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=has no field 'z'
struct Point(val x: i64, val y: i64)

fn main(): void {
    val p = Point(x: 10, z: 20)
}
