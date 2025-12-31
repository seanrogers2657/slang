// @test: exit_code=0
// @test: stdout=true\ntrue\ntrue\nfalse\n
// Tests logical OR operator truth table
main = () {
    print(true || true)   // true
    print(true || false)  // true
    print(false || true)  // true
    print(false || false) // false
}
