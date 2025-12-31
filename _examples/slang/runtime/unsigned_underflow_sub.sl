// @test: exit_code=1
// @test: stderr_contains=panic: unsigned underflow: subtraction
main = () {
    val zero: u64 = 0
    val one: u64 = 1
    val result = zero - one
    print(result)
}
