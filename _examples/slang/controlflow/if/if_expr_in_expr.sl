// @test: exit_code=15
// Tests if expression used within another expression
main = () {
    val x = 5
    val result = 10 + if x > 3 { 5 } else { 0 }
    exit(result)
}
