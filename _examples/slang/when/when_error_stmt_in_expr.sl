// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=must contain expressions, not statements
main = () {
    var x = 0
    val result = when {
        true -> x = 42
        else -> 0
    }
    exit(result)
}
