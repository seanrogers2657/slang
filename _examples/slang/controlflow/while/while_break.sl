// @test: exit_code=0
// @test: stdout=0\n1\n2\n
main = () {
    var i = 0
    while i < 10 {
        if i == 3 {
            break
        }
        print(i)
        i = i + 1
    }
}
