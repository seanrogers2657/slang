// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=moved value
// Test: Move in any branch invalidates variable after if/else
Point = struct {
    val x: s64
    val y: s64
}

main = () {
    val p = new Point{ 1, 2 }

    if true {
        val q = p  // moves p in this branch
    }

    print(p.x)  // Error: p may have been moved
}
