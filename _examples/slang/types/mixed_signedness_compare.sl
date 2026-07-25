// @test: exit_code=0
// @test: stdout=true\ntrue\ntrue\nfalse\ntrue\nfalse\ntrue\ntrue\ntrue\n
// When an unsigned operand widens into a signed one, the comparison must be
// evaluated as SIGNED. Reading it as unsigned would treat a negative signed
// operand as a huge positive value and invert every result.
main = () {
    val n: s64 = -1
    val b: u8 = 1

    print(b > n)      // 1 > -1
    print(n < b)      // -1 < 1
    print(b >= n)
    print(n >= b)
    print(b != n)
    print(b == n)

    // Wider unsigned types widen the same way
    val w: u32 = 5
    print(w > n)

    // Both operands unsigned still compares as UNSIGNED: u64 max must read
    // as the largest value, not as -1
    val big: u64 = 18446744073709551615
    val one: u64 = 1
    print(big > one)

    // Both operands signed is unaffected
    val z: s64 = 0
    print(z > n)
}
