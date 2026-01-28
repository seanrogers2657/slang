// @test: exit_code=0
// Test assert with variables and expressions
main = () {
    val x = 10
    val y = 5
    assert(x > y, "x should be greater than y")
    assert(x + y == 15, "sum should be 15")
    assert(x - y == 5, "difference should be 5")

    var count = 0
    count = count + 1
    assert(count == 1, "count should be 1 after increment")
}
