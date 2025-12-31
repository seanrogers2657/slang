// @test: exit_code=0
// @test: stdout=22\n
double = (x: int) -> int {
    return x * 2
}

add_one = (x: int) -> int {
    return x + 1
}

main = () {
    val result = double(add_one(double(5)))
    print(result)
}
