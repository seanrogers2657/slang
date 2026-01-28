// @test: exit_code=42
// Test deeply nested if statements

main = () {
    val a = 10
    val b = 20
    val c = 30
    val d = 40

    if a < b {
        if b < c {
            if c < d {
                if d > a {
                    exit(42)  // All conditions true
                }
            }
        }
    }
    exit(0)
}
