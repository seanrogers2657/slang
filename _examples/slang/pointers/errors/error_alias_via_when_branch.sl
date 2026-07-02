// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot bind owned value
// Regression: a when-expression branch that yields an existing owner must be
// rejected like a direct alias, in both the bare-expression and block-body
// branch forms.
Point = struct { var x: s64  var y: s64 }

main = () {
    val p = new Point{ 1, 2 }
    val q = when {
        else -> p   // Error: aliases p through the branch
    }
    print(q.x)
}
