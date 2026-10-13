// @test: stdout=170141183460469231731687303715884105727\n255\n8\n-128\ntrue\nfalse\n
// A suffix gives an integer literal its own type, so it needs no declared
// type in scope to be inferred from.
show = (x: s128) {
    print(x)
}

main = () {
    // A 128-bit literal passed directly as an argument
    show(170141183460469231731687303715884105727s128)

    // An unannotated binding takes the suffix's type
    val a = 200u8
    print(a + 55u8)

    // A suffixed literal drives coercion of a bare literal like any operand
    print(5 + 3u8)

    // The minimum of a signed type, whose magnitude alone is out of range
    print(-128s8)

    val m = 18446744073709551615u64
    print(m == 18446744073709551615u64)

    val n: u32? = 4294967295u32
    print(n == null)
}
