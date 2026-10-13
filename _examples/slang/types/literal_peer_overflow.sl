// @test: exit_code=1
// @test: stderr_contains=panic: unsigned overflow: addition
// The if-expression is u8 (from its suffixed branch), so arithmetic on it
// traps at u8 width rather than widening.
main = () {
    val x = if true { 250u8 } else { 1 }
    print(x + 10)
}
