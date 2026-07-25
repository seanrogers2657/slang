// @test: exit_code=0
// @test: stdout=0123456789\n10\n
// Accumulating into a var string in a loop reassigns the binding each
// iteration; the previous buffer is freed by the reassignment, so the heap
// stays balanced at exit.
main = () {
    var s = ""
    for (var i = 0; i < 10; i = i + 1) {
        s = s + "${i}"
    }
    print(s)
    print(len(s))
}
