// @test: stdout=200 -5 65535 -32768\n4294967295 -2147483648\n18446744073709551615 9223372036854775808 null\n200+255\nbyte=104\n
// Every integer width interpolates, signed and unsigned, including u64
// values above the s64 range and nullable integers.
main = () {
    val a = 200u8
    val b = -5s8
    val c = 65535u16
    val d = -32768s16
    val e = 4294967295u32
    val f = -2147483648s32
    val g = 18446744073709551615u64
    val h: u64? = 9223372036854775808u64
    val n: u8? = null
    print("${a} ${b} ${c} ${d}")
    print("${e} ${f}")
    print("${g} ${h} ${n}")
    print("$a+${a + 55u8}")
    val s = "hi"
    print("byte=${s[0]}")
}
