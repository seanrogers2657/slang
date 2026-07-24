// @test: exit_code=0
// @test: stdout=340282366920938463463374607431768211455\n18446744073709551616\n0\n170141183460469231731687303715884105727\n1000000000000000000\n333\n1\n
// True 128-bit unsigned arithmetic and unsigned wraparound, division, modulo.
main = () {
    val max: u128 = 340282366920938463463374607431768211455    // 2^128 - 1
    print(max)

    val two64: u128 = 18446744073709551616                     // 2^64
    print(two64)
    print(two64 * two64)    // 2^128 wraps to 0

    val two: u128 = 2
    print(max / two)        // (2^128 - 1) / 2 = 2^127 - 1

    val big: u128 = 1000000000000000000000000000000000000       // 10^36
    val ten9: u128 = 1000000000000000000                        // 10^18
    print(big / ten9)       // 10^18

    val d: u128 = 1000
    val e: u128 = 3
    print(d / e)            // 333
    print(d % e)            // 1
}
