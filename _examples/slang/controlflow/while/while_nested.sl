// @test: exit_code=0
// @test: stdout=0\n1\n2\n3\n4\n5\n
main = () {
    var i = 0
    while i < 3 {
        var j = 0
        while j < 2 {
            print(i * 2 + j)
            j = j + 1
        }
        i = i + 1
    }
}
