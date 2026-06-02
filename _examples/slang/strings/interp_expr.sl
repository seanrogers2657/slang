// @test: exit_code=0
// @test: stdout=2 + 3 = 5\n10 > 3 is true\n
// Interpolation of arbitrary expressions inside ${...}.
main = () {
    val a = 2
    val b = 3
    print("${a} + ${b} = ${a + b}")
    print("10 > 3 is ${10 > 3}")
}
