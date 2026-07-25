// @test: exit_code=0
// @test: stdout=hello\nworldtest\n
// Escape sequences survive concatenation
main = () {
    print("hello\nworld" + "test")
}
