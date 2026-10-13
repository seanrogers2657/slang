// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=array element type mismatch: expected u8, got s64
// A literal that does not fit the typed peer is not truncated into it.
main = () {
    val b: u8 = 7
    val a = [b, 300]
    print(a[0])
}
