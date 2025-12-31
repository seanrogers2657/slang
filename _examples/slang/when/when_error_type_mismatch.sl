// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=different types
main = () {
    val result = when {
        true -> 42
        else -> "hello"
    }
    exit(0)
}
