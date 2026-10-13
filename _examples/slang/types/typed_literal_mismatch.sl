// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot assign s64 to variable of type u8
// A suffixed literal has a fixed type: it is not retyped to the annotation.
main = () {
    val x: u8 = 5s64
    print(x)
}
