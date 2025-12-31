// @test: exit_code=0
// @test: stdout=10\n20\n200\n
add = (a: int, b: int) -> int {
    return a + b
}

multiply = (a: int, b: int) -> int {
    return a * b
}

main = () {
    val x = add(3, 7)
    print(x)
    val y = multiply(2, 10)
    print(y)
    val z = multiply(x, y)
    print(z)
}
