// @test: exit_code=15
fn main(): void {
    val x = 5
    val result = when {
        x > 3 -> {
            val temp = x * 2
            temp + 5
        }
        else -> 0
    }
    exit(result)
}
