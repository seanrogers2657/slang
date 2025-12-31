// @test: exit_code=42
// Tests nested when expressions
main = () {
    val x = 5
    val y = 10
    val result = when {
        x > 3 -> when {
            y > 15 -> 100
            y > 5 -> 42
            else -> 0
        }
        else -> 1
    }
    exit(result)
}
