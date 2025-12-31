// @test: exit_code=0
// @test: stdout=0\n1\n2\n
main = () {
    for var i = 0; i < 3; i = i + 1 {
        print(i)
    }
}
