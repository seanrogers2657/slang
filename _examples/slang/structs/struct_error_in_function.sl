// @test: expect_error=true
// @test: error_stage=parser
// @test: error_contains=struct declarations are only allowed at the top level
fn main(): void {
    struct BadStruct(val x: i64)
    val p = BadStruct(10)
    print(p.x)
}
