// @test: exit_code=1
// Tests if with logical operators
main = () {
    val x = 5
    val y = 10
    if x > 0 && y > 0 {
        exit(1)
    } else {
        exit(2)
    }
}
