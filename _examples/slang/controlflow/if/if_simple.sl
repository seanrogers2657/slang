// @test: exit_code=42
// Tests basic if statement (condition true)
main = () {
    if true {
        exit(42)
    }
    exit(0)
}
