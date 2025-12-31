// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=has no field 'z'
struct Point(val x: i64, val y: i64)

main = () {
    val p = Point(1, 2)
    print(p.z)
}
