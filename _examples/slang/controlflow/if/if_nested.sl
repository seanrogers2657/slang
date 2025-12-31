// @test: exit_code=42
// Tests nested if statements
main = () {
    val x = 10
    val y = 20
    if x > 5 {
        if y > 15 {
            exit(42)
        } else {
            exit(1)
        }
    } else {
        exit(2)
    }
}
