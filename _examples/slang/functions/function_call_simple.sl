// @test: exit_code=0
// @test: stdout=42\n
get_value = () -> int {
    return 42
}

main = () {
    val result = get_value()
    print(result)
}
