// @test: exit_code=0
// Test that assert passes when condition is true
main = () {
    assert(true, "this should not fail")
    assert(1 == 1, "equality check")
    assert(5 > 3, "comparison check")
}
