// @test: exit_code=0
// @test: stdout=x is null\ny is 99\n
// Interpolating nullable values: null renders as "null", otherwise the value.
main = () {
    val x: s64? = null
    print("x is ${x}")
    val y: s64? = 99
    print("y is ${y}")
}
