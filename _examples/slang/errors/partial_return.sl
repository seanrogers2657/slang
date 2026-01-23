// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=does not return a value on all code paths
getValue = (cond: bool) -> s64 {
    if cond {
        return 42
    }
    // missing else branch with return
}

main = () {
    print(getValue(true))
}
