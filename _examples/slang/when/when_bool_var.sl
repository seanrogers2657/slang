// @test: exit_code=42
// Tests when with boolean variable in condition
main = () {
    val is_large = false
    val is_small = true
    val result = when {
        is_large -> 100
        is_small -> 42
        else -> 0
    }
    exit(result)
}
