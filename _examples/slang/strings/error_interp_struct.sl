// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot interpolate value of type
// Interpolating a non-supported type (a struct) is a compile error.
Point = struct {
    val x: s64
}

main = () {
    val p = Point{ 1 }
    print("p = ${p}")
}
