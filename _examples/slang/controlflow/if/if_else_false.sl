// @test: exit_code=2
// Tests if-else when condition is false
main = () {
    val x = 3
    if x > 5 {
        exit(1)
    } else {
        exit(2)
    }
}
