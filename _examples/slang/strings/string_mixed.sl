// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot concatenate 'string' and 's64'
main = () {
    "value" + 42
}
