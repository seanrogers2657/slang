// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=undefined variable
// A binding declared in a bare block is scoped to that block and cannot be
// referenced after the closing brace.
main = () {
    {
        val secret = 5
        print(secret)
    }
    print(secret)
}
