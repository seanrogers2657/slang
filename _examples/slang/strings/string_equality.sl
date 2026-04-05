// @test: exit_code=0
// @test: stdout=true\ntrue\nfalse\ntrue\nfalse\n
// String equality and inequality comparisons

main = () {
    val a = "hello"
    val b = "hello"
    val c = "world"

    print(a == b)
    print(a != c)
    print(a == c)
    print("foo" == "foo")
    print("foo" == "bar")
}
