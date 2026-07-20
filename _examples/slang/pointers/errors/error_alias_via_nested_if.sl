// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot bind owned value
// Regression: a nested if-expression ending a branch block is a statement node,
// not an expression statement — the alias check must still recurse into its
// branch results, or an owner alias hides one level down and double-frees.
Point = struct { var x: s64  var y: s64 }

main = () {
    val p = new Point{ 1, 2 }
    val q = if true {
        if true { p } else { p }   // Error: aliases p through nested branches
    } else {
        new Point{ 3, 4 }
    }
    print(q.x)
}
