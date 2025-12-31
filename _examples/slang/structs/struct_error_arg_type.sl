// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=field 'x': expected i64, got boolean
struct Point(val x: i64, val y: i64)

main = () {
    val p = Point(true, 2)
}
