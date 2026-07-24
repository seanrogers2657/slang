// @test: exit_code=0
// @test: stdout=36893488147419103232\n340282366920938463463374607431768211455\n-5\n10\n
// 128-bit values flow through function parameters/returns, struct fields, and
// loop counters (the multi-word value is moved word-by-word by the backend).
Pair = struct { val big: u128  val small: s128 }

add_u128 = (a: u128, b: u128) -> u128 {
    return a + b
}

main = () {
    val x: u128 = 18446744073709551616
    print(add_u128(x, x))     // 2^65

    val p = Pair{ 340282366920938463463374607431768211455, -5 }
    print(p.big)              // u128 max (a field-stored 128-bit literal)
    print(p.small)            // -5

    var i: u128 = 0
    var sum: u128 = 0
    val one: u128 = 1
    val limit: u128 = 5
    while i < limit {
        sum = sum + i
        i = i + one
    }
    print(sum)                // 0+1+2+3+4 = 10
}
