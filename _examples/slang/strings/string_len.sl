// @test: exit_code=0
// @test: stdout=5\n0\n13\n
// len(string) returns the number of bytes in the string
main = () {
    print(len("hello"))
    print(len(""))
    print(len("hello, world!"))
}
