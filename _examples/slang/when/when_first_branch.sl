// @test: exit_code=1
// Tests that the first matching branch is taken
main = () {
    val x = 100
    val result = when {
        x > 50 -> 1
        x > 10 -> 2
        else -> 3
    }
    exit(result)
}
