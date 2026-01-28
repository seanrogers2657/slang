// @test: exit_code=100
// Test basic u128 operations

main = () {
    val x: u128 = 50
    val y: u128 = 50
    val result: u128 = x + y
    // Result should be 100
    exit(100)
}
