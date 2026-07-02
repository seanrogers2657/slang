// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot bind owned value
// Regression: an elvis operand that names an existing owner must be rejected —
// either operand may become the stored value, so `p ?: q` can alias an owner
// just like a direct binding.
Point = struct { var x: s64  var y: s64 }

main = () {
    val p: *Point? = new Point{ 1, 2 }
    val q = p ?: new Point{ 9, 9 }   // Error: left operand aliases p
    print(q.x)
}
