// @test: stdout=hello\n
fn main(): void {
    val x = 5
    when {
        x > 10 -> print("big")
        x > 3 -> print("hello")
        else -> print("small")
    }
}
