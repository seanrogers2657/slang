// @test: stdout=hello\n
main = () {
    val x = 5
    when {
        x > 10 -> print("big")
        x > 3 -> print("hello")
        else -> print("small")
    }
}
