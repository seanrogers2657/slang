// @test: exit_code=1
// @test: stderr=panic: chr value out of byte range\nat main()\n
// A negative value has no byte representation either
main = () {
    val n: s64 = -191
    print(chr(n))
}
