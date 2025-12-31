// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot assign to immutable field
struct Point(val x: i64, val y: i64)

main = () {
    val p = Point(10, 20)
    p.x = 30
}
