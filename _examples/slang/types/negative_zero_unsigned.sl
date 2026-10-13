// @test: stdout=0\n0\ntrue\n0\n
// Negating a zero literal yields plain zero, which is valid for unsigned types.
main = () {
    val a: u8 = -0
    val b: u64 = -000
    print(a)
    print(b)
    print(-0u8 == 0u8)
    print(-0u128)
}
