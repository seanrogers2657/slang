// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=return type mismatch
get_number = () -> int {
    return "hello"
}

main = () {
    val x = get_number()
}
