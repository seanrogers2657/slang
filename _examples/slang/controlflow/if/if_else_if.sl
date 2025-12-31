// @test: exit_code=2
// Tests else-if chaining
main = () {
    val x = 5
    if x > 10 {
        exit(1)
    } else if x == 5 {
        exit(2)
    } else {
        exit(3)
    }
}
