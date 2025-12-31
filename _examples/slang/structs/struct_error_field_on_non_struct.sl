// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot access field 'value' on non-struct type
main = () {
    val x = 42
    print(x.value)
}
