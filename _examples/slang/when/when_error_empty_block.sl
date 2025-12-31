// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot be empty
main = () {
    val result = when {
        true -> {
        }
        else -> 0
    }
    exit(result)
}
