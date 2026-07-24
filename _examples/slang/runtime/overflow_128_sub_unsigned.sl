// @test: exit_code=1
// @test: stderr_contains=panic: unsigned underflow: subtraction
// 128-bit unsigned subtraction traps on borrow-out (underflow below zero).
main = () {
    val zero: u128 = 0
    val one: u128 = 1
    print(zero - one)
}
