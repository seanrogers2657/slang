// @test: exit_code=0
// @test: stdout=1\n1\n1\n
// Regression: a vec yielded from an if/when branch is copied at the branch
// (value semantics) so the new binding owns an independent buffer. Aliasing
// the source vec used to double-free — both bindings freed the same header.
main = () {
    var v1 = vec()
    push(v1, 1)
    val c = true

    val v2 = if c { v1 } else { vec() }
    print(len(v2))   // 1 — independent copy of v1

    val v3 = when {
        c -> v1
        else -> vec()
    }
    print(len(v3))   // 1

    push(v2, 2)
    print(len(v1))   // 1 — v1 unaffected by v2's growth
}
