// @test: exit_code=1
// @test: stderr=panic: chr value out of byte range\nat main()\n
// chr traps on a value with no byte representation rather than silently
// keeping the low 8 bits (321 & 0xFF would otherwise yield "A")
main = () {
    val n: s64 = 321
    print(chr(n))
}
