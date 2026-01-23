// @test: exit_code=0
// @test: stdout=true\nfalse\nfalse\ntrue\n
// Test reassigning mutable nullable variables
main = () {
    var x: s64? = 42
    print(x != null)  // true - has value

    x = null
    print(x != null)  // false - now null

    x = 100
    print(x == null)  // false - has value again

    x = null
    print(x == null)  // true - null again
}
