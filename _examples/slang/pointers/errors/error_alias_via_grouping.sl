// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot bind owned value
// Regression: wrapping an owner in parentheses must not smuggle an alias past
// the single-owner check. `val q = (p)` is the same double-free as `val q = p`.
Point = struct { var x: s64  var y: s64 }

main = () {
    val p = new Point{ 1, 2 }
    val q = (p)   // Error: still an alias of p
    print(q.x)
}
