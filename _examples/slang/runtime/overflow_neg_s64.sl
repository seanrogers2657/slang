// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: negation
// Regression: negating s64 INT_MIN wraps back to INT_MIN (the true result +2^63
// does not fit), so unary negation needs a full-width overflow trap.
main = () {
    var a: s64 = -9223372036854775807
    a = a - 1        // s64 min
    print(-a)
}
