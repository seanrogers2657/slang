// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=does not return a value on all code paths
getValue = () -> i64 {
    val x = 42
}

main = () {
    print(getValue())
}
