// @test: exit_code=1
// @test: stderr_contains=panic: unsigned overflow: addition
// Arithmetic on suffixed literals stays at the suffix's width and traps.
main = () {
    val x = 250u8
    print(x + 10u8)
}
