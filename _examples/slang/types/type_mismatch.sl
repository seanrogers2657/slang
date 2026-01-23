// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=requires operands of the same type
main = () {
    val x: s32 = 10
    val y: s64 = 20
    val z = x + y
}
