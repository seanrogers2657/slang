// @test: exit_code=0
// @test: stdout=8\n15\n
add = (a: int, b: int) -> int {
    return a + b
}

main = () {
    val sum1 = add(3, 5)
    print(sum1)
    val sum2 = add(7, 8)
    print(sum2)
}
