// @test: exit_code=0
// @test: stdout=170141183460469231731687303715884105726\n-170141183460469231731687303715884105728\n-170141183460469231731687303715884105728\n1000000000000000000000000000000000000\n
// In-range 128-bit results at the boundary must NOT falsely trap: max-1,
// min*1, and -2^63 * 2^64 = -2^127 (exactly INT_MIN) all fit.
main = () {
    val max: s128 = 170141183460469231731687303715884105727
    val one: s128 = 1
    print(max - one)         // fits
    print(min_times_one())   // min * 1 = min, magnitude 2^127 but negative -> fits
    val n63: s128 = -9223372036854775808
    val p64: s128 = 18446744073709551616
    print(n63 * p64)         // -2^127 = INT_MIN, fits
    val a: s128 = -1000000000000000000
    print(a * a)             // 10^36, fits
}

min_times_one = () -> s128 {
    val min: s128 = -170141183460469231731687303715884105728
    val one: s128 = 1
    return min * one
}
