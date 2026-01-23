// @test: exit_code=120
// Test recursive method with method call directly as operand

Calculator = class {
    var result: i64

    // Recursive static method - factorial with direct operand usage
    factorial = (n: i64) -> i64 {
        if n <= 1 {
            return 1
        }
        return n * Calculator.factorial(n - 1)
    }
}

main = () {
    val result = Calculator.factorial(5)  // 5! = 120
    exit(result)
}
