// @test: exit_code=0
// @test: stdout=s128 max is 170141183460469231731687303715884105727\nneg = -42\nu128 max is 340282366920938463463374607431768211455\nsum = 3000000000000000000000\nmaybe = 42 or null\n
// String interpolation renders 128-bit integers at full width (via the
// _sl_{u,s}128_to_str heap-string helpers), including interpolated expressions
// and nullable forms.
main = () {
    val x: s128 = 170141183460469231731687303715884105727
    print("s128 max is $x")
    val n: s128 = -42
    print("neg = $n")
    val u: u128 = 340282366920938463463374607431768211455
    print("u128 max is $u")

    val a: u128 = 1000000000000000000000
    val b: u128 = 2000000000000000000000
    print("sum = ${a + b}")

    val some: s128? = 42
    val none: s128? = null
    print("maybe = $some or $none")
}
