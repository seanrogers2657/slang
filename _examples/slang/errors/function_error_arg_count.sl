// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=expects 2 arguments
add = (a: int, b: int) -> int {
    return a + b
}

main = () {
    val x = add(1)
}
