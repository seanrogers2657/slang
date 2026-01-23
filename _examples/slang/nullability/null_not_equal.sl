// @test: exit_code=0
// @test: stdout=false\ntrue\n
// Test != null comparison
main = () {
    val x: s64? = null
    val y: s64? = 42
    print(x != null)
    print(y != null)
}
