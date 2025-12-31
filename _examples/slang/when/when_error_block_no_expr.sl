// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=must end with an expression
main = () {
    var x = 0
    val result = when {
        true -> {
            x = 42
        }
        else -> 0
    }
    exit(result)
}
