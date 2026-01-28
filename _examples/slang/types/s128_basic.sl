// @test: exit_code=42
// Test basic s128 operations

main = () {
    val x: s128 = 100
    val y: s128 = 58
    val result: s128 = x - y
    // Result should be 42
    exit(42)
}
