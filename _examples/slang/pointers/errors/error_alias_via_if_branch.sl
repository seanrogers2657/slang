// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot bind owned value
// Regression: an if-expression branch that yields an existing owner must be
// rejected like a direct alias — the branch result reaches the new binding, so
// both bindings would free the same allocation. A branch may yield a fresh
// value (new/.copy()) instead.
Point = struct { var x: s64  var y: s64 }

main = () {
    val p = new Point{ 1, 2 }
    val q = if true { p } else { p }   // Error: aliases p through the branches
    print(q.x)
}
