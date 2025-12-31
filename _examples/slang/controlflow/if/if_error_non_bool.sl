// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=must be boolean
// Tests that non-boolean condition produces error
main = () {
    if 42 {
        exit(1)
    }
}
