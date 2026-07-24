// @test: exit_code=1
// @test: stderr_contains=panic: unsigned overflow: multiplication
// 128-bit unsigned multiplication traps when the product exceeds 128 bits.
main = () {
    val two64: u128 = 18446744073709551616
    print(two64 * two64)   // 2^128
}
