// @test: exit_code=3
// Test elvis operator precedence - should be lower than arithmetic
// a ?: b + c should parse as a ?: (b + c), not (a ?: b) + c

main = () {
    // Elvis with non-null - should use original value, not add
    val y: s64? = 3
    val r2 = y ?: 5 + 10  // Should be: 3 (not 3 + 10 = 13)

    // If precedence is correct: r2 = 3
    // If precedence is wrong (?: binds tighter than +): r2 = 13

    exit(r2)  // Should be 3
}
