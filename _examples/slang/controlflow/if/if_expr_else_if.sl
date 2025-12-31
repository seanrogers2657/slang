// @test: exit_code=2
// Tests if expression with else-if chain
main = () {
    val x = 5
    val result = if x > 10 { 1 } else if x > 3 { 2 } else { 3 }
    exit(result)
}
