// @test: exit_code=42
// Tests nested if expressions
main = () {
    val x = 10
    val y = 5
    val result = if x > 5 {
        if y > 3 { 42 } else { 1 }
    } else {
        0
    }
    exit(result)
}
