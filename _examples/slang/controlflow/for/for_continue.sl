// @test: exit_code=0
// @test: stdout=1\n3\n5\n
main = () {
    for (var i = 0; i < 6; i = i + 1) {
        if i % 2 == 0 {
            continue
        }
        print(i)
    }
}
