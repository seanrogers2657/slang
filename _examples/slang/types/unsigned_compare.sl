// @test: exit_code=0
// @test: stdout=true\ntrue\ntrue\ntrue\n
// Regression: relational comparisons of unsigned integers must use unsigned
// condition codes. A u64 above 2^63 was compared as a negative signed value,
// so `big > small` was false.
main = () {
    val big: u64 = 18446744073709551615
    val small: u64 = 1
    print(big > small)    // true
    print(small < big)    // true
    print(big >= small)   // true
    print(small <= big)   // true
}
