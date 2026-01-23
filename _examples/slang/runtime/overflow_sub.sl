// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: subtraction
main = () {
    // Compute MIN_I64 step by step to avoid negative literals
    var min: s64 = 0
    min = min - 9223372036854775807
    min = min - 1
    // Now min = -9223372036854775808 (MIN_I64)
    val one: s64 = 1
    val c = min - one
    print(c)
}
