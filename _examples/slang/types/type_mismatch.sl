// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=requires operands of the same type
// Different signedness still causes type mismatch
main = () {
    val x: s32 = 10
    val y: u32 = 20
    val z = x + y
}
