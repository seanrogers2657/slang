// @test: exit_code=0
// @test: stdout=7\n6\n5\n
// A name may be reused in sibling (non-overlapping) scopes, and a local may
// share a name with a top-level function — neither shadows an enclosing local,
// so both are allowed. Only reuse across a live enclosing scope is rejected.
helper = () -> s64 { return 99 }

main = () {
    // Sibling while loops each declare `step` — non-overlapping scopes.
    var a = 0
    var i = 0
    while i < 3 {
        val step = 1
        a = a + step
        i = i + 1
    }
    var j = 0
    while j < 2 {
        val step = 2
        a = a + step
        j = j + 1
    }
    print(a)              // 7

    // Nested loops with distinct counter names.
    var sum = 0
    for (var m = 0; m < 3; m = m + 1) {
        for (var n = 0; n < 2; n = n + 1) {
            sum = sum + 1
        }
    }
    print(sum)            // 6

    // A local may shadow a top-level function name (different SSA space).
    val helper = 5
    print(helper)         // 5
}
