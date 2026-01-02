// @test: exit_code=0
// @test: stdout=false\nfalse\n
// Test that null stays null after a function call
dummy = () {
    // do nothing
}

main = () {
    val y: i64? = null
    print(y != null)  // false
    dummy()
    print(y != null)  // should still be false
}
