// @test: exit_code=0
// @test: stdout=true\nfalse\n
// Regression: an if-expression whose branch ends in a short-circuit && / ||
// must yield the &&/|| result. The short-circuit constant used to be emitted
// after the merge phi, so getLastValue() picked up the constant instead of the
// phi and the if-expression evaluated to the wrong value.
main = () {
    val a = 3
    val b = 7
    val r1 = if a > 0 { (a > 1 && b > 5) } else { false }
    print(r1)  // true

    val r2 = if a > 0 { (a > 5 || b > 100) } else { true }
    print(r2)  // false
}
