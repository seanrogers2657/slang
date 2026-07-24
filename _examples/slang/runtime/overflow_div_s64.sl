// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: division
// Regression: signed INT_MIN / -1 overflows (the true result is +2^63, which
// does not fit in s64). sdiv wraps it back to INT_MIN silently, so division
// needs an explicit full-width overflow trap like add/sub/mul.
main = () {
    var a: s64 = -9223372036854775807
    a = a - 1        // s64 min
    var b: s64 = -1
    print(a / b)
}
