// @test: exit_code=0
// @test: stdout=-100\ntrue\n
// Regression: loading a narrow signed field (s8/s16/s32) must sign-extend.
// Byte/half/word loads zero-extend, so a stored negative s8 read back as a
// large positive value (156 instead of -100).
Box = struct {
    var v: s8
}

main = () {
    var b = Box{ 0 }
    b.v = -100
    print(b.v)         // -100
    print(b.v < 0)     // true
}
