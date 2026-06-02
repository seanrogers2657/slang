// @test: exit_code=0
// @test: stdout=ab\nWorldWorld\n
// Adjacent interpolations with empty literal chunks between them.
main = () {
    val a = "a"
    val b = "b"
    print("${a}${b}")
    val w = "World"
    print("$w$w")
}
