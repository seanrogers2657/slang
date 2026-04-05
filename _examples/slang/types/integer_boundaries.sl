// @test: exit_code=0
// @test: stdout=127\n255\n32767\n65535\n
// Test integer type boundaries and implicit widening for comparisons

main = () {
    // s8 max: 127
    val s8_max: s8 = 127
    assert(s8_max == 127, "s8 max should be 127")
    print(s8_max)

    // u8 max: 255 (compared with u8 literal since u8 != s64 signedness)
    val u8_max: u8 = 255
    val u8_cmp: u8 = 255
    assert(u8_max == u8_cmp, "u8 max should be 255")
    print(u8_max)

    // s16 max: 32767
    val s16_max: s16 = 32767
    assert(s16_max == 32767, "s16 max should be 32767")
    print(s16_max)

    // u16 max: 65535
    val u16_max: u16 = 65535
    val u16_cmp: u16 = 65535
    assert(u16_max == u16_cmp, "u16 max should be 65535")
    print(u16_max)

    // Arithmetic at boundaries (staying in range)
    val near_max: s8 = 126
    val one: s8 = 1
    val at_max: s8 = near_max + one
    assert(at_max == 127, "126 + 1 should be 127")
}
