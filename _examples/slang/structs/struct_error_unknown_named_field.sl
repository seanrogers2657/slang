// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=has no field 'z'
Point = struct {
    val x: s64
    val y: s64
}

main = () {
    val p = Point{ x: 10, z: 20 }
}
