// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains='continue' statement not inside a loop
main = () {
    continue
}
