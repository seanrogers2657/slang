// @test: exit_code=0
// @test: stdout=0\n0\n0\n1\n1\n0\n1\n1\n
main = () {
    for (var i = 0; i < 2; i = i + 1) {
        for (var j = 0; j < 2; j = j + 1) {
            print(i)
            print(j)
        }
    }
}
