// @test: exit_code=0
// @test: stdout=2\n1\n
// Regression: phi-node copies on a loop back-edge that swap two variables must
// be resolved as a parallel copy. A naive sequential lowering clobbered one
// value and printed "2\n2" instead of "2\n1".
main = () {
    var a = 1
    var b = 2
    var i = 0
    while i < 1 {
        val t = a
        a = b
        b = t
        i = i + 1
    }
    print(a)
    print(b)
}
