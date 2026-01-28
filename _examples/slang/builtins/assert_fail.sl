// @test: exit_code=1
// @test: stderr_contains=assertion failed: expected 1 to equal 2
// Test that assert fails when condition is false
main = () {
    assert(false, "expected 1 to equal 2")
}
