// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot assign null to non-nullable
// Test that returning null from a non-nullable return type is an error
getValue = () -> s64 {
    return null
}

main = () {
    val x = getValue()
}
