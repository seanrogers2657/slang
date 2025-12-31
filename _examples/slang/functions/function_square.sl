// @test: exit_code=0
// @test: stdout=25\n
square = (n: int) -> int {
    return n * n
}

main = () {
    val result = square(5)
    print(result)
}
