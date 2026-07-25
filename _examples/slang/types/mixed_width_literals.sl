// @test: exit_code=0
// @test: stdout=true\ntrue\n5\n1200\nfalse\n255\n65535\n4294967295\n100\n
// A bare integer literal takes the type of the other operand, so narrow
// values can be compared and combined with literals without a conversion.
main = () {
    val b: u8 = 200

    // Literal coerces to u8 on both sides of a comparison
    print(b > 100)
    print(100 < b)

    // Literal coerces to u8 in arithmetic
    val small: u8 = 3
    print(small + 2)

    // A literal too large for the narrow operand is not coerced down (that
    // would truncate); instead the narrow side widens and the operation is
    // evaluated exactly at the wider type
    print(b + 1000)
    print(b > 300)

    // Unsigned values widen to the default s64 because they are held
    // zero-extended, so the full unsigned range survives the widening
    val z: s64 = 0
    val m8: u8 = 255
    val m16: u16 = 65535
    val m32: u32 = 4294967295
    print(z + m8)
    print(z + m16)
    print(z + m32)

    // Narrow arithmetic stays narrow and unsigned
    var d: u8 = 200
    d = d - 100
    print(z + d)
}
