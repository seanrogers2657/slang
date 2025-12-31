// @test: exit_code=42
// Tests basic if expression assignment
main = () {
    val x = if true { 42 } else { 0 }
    exit(x)
}
