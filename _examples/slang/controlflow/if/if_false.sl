// @test: exit_code=0
// Tests basic if statement (condition false)
main = () {
    if false {
        exit(42)
    }
    exit(0)
}
