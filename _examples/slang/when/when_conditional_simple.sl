// @test: exit_code=10
main = () {
    val x = 5
    when {
        x > 10 -> exit(0)
        x > 3 -> exit(10)
        else -> exit(20)
    }
}
