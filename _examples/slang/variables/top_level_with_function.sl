// @test: stdout=100\n
double = (x: s64) -> s64 {
    return x + x
}

val result = double(50)

main = () {
    print(result)
}
