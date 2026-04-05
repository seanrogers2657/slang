// @test: exit_code=0
// @test: stdout=42\n99\n7\n0\n
// Elvis chaining with nullable function returns

find_positive = (x: s64) -> s64? {
    if x > 0 { return x }
    return null
}

main = () {
    // Elvis chain: first non-null wins
    val result = find_positive(0) ?: find_positive(0) ?: find_positive(42) ?: 0
    assert(result == 42, "should find 42")
    print(result)

    // All null -> default
    val fallback = find_positive(0) ?: find_positive(0) ?: find_positive(0) ?: 99
    assert(fallback == 99, "should fall through to 99")
    print(fallback)

    // First value wins
    val first = find_positive(7) ?: find_positive(99) ?: 0
    assert(first == 7, "should find 7 first")
    print(first)

    // Single elvis
    val single = find_positive(0) ?: 0
    assert(single == 0, "null should give default 0")
    print(single)
}
