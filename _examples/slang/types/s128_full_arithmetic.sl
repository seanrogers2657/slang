// @test: exit_code=0
// @test: stdout=170141183460469231731687303715884105727\n-170141183460469231731687303715884105728\n36893488147419103232\n-36893488147419103232\n-142857142857142857142\n-6\n
// True 128-bit signed arithmetic: values beyond 64 bits are stored in two
// words, computed with hand-propagated carry/borrow, and printed at full width
// via the software divmod-by-10.
main = () {
    val max: s128 = 170141183460469231731687303715884105727    // 2^127 - 1
    val min: s128 = -170141183460469231731687303715884105728   // -2^127
    print(max)
    print(min)

    val two64: s128 = 18446744073709551616                     // 2^64
    val two: s128 = 2
    print(two64 * two)      // 2^65 = 36893488147419103232
    print(-(two64 * two))   // -36893488147419103232

    val a: s128 = -1000000000000000000000
    val b: s128 = 7
    print(a / b)            // -142857142857142857142 (truncated)
    print(a % b)            // -6 (remainder takes dividend's sign)
}
