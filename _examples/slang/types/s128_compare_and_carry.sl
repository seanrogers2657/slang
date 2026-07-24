// @test: exit_code=0
// @test: stdout=true\ntrue\ntrue\ntrue\ntrue\ntrue\ntrue\n
// 128-bit comparisons decide on the high word first (signed for s128, unsigned
// for u128) then the low word, and carry/borrow cross the 64-bit boundary.
main = () {
    val one: u128 = 1
    val two: u128 = 2
    val max64: u128 = 18446744073709551615     // 2^64 - 1
    val two64: u128 = 18446744073709551616      // 2^64
    print(max64 + one == two64)   // carry into high word
    print(two64 - one == max64)   // borrow from high word
    print(two64 > max64)          // unsigned high-word ordering
    print(two64 * two > two64)    // 2^65 > 2^64

    val min: s128 = -170141183460469231731687303715884105728
    val neg1: s128 = -1
    val zero: s128 = 0
    print(min < neg1)             // signed high-word ordering
    print(neg1 < zero)
    print(min < zero)
}
