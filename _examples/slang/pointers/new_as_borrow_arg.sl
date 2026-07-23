// @test: exit_code=0
// @test: stdout=7\n1\n
// Regression: `new T{...}` passed directly as a borrow argument (or as a
// bare discarded statement) is an allocation no binding owns; the caller
// must free it after the call. It leaked one allocation per use.
Point = struct { var x: s64  var y: s64 }

sum_pt = (p: &Point) -> s64 {
    return p.x + p.y
}

main = () {
    print(sum_pt(new Point{ 3, 4 }))   // 7 — temp freed after the call
    new Point{ 9, 9 }                  // discarded: freed, not leaked
    print(1)
}
