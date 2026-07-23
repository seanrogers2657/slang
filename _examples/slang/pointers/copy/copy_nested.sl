// @test: exit_code=0
// @test: stdout=5\n1\n99\n
// Regression: .copy() must recurse into a nested aggregate VALUE field so the
// copy owns independent storage at every level. Mutating the original's nested
// field after the copy must not be visible through the copy.
Inner = struct { var cost: s64 }
Outer = struct {
    var inner: Inner
    var tag: s64
}

main = () {
    var p = new Outer{ Inner{ 5 }, 1 }
    val q = p.copy()       // deep copy: recurses into the nested Inner field

    p.inner.cost = 99      // mutate the original's nested field
    p.tag = 88

    print(q.inner.cost)    // 5  — copy is independent at the nested level
    print(q.tag)           // 1
    print(p.inner.cost)    // 99 — original sees its own mutation
}
