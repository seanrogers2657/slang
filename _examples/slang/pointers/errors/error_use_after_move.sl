// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot bind owned value
// Test: Binding an owned pointer to another variable is rejected
//       (there can be only one owner).
Point = struct {
    val x: s64
    val y: s64
}

main = () {
    val p = new Point{ 10, 20 }
    val q = p  // Error: cannot bind owned value 'p' to another variable
    print(p.x)
}
