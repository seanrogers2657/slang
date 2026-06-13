// @test: exit_code=0
// @test: stdout=18446744073709551615\n4294967295\n255\n
// Regression: print() of an unsigned integer must not interpret the value as
// signed. A u64 above 2^63 used to print as a negative number (e.g. -1).
main = () {
    val a: u64 = 18446744073709551615
    print(a)

    val b: u32 = 4294967295
    print(b)

    val c: u8 = 255
    print(c)
}
