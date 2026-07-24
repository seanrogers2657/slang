// @test: exit_code=0
// @test: stdout=0\n1\n2\n99\n1\n3\n
// Regression: break/continue used as a when-case body inside a loop must
// compile (previously hung the parser) and produce the right control flow.
main = () {
    var i = 0
    while true {
        when {
            i >= 3 -> break
            else -> print(i)
        }
        i = i + 1
    }
    print(99)

    for (var j = 0; j < 5; j = j + 1) {
        when {
            j % 2 == 0 -> continue
            else -> print(j)
        }
    }
}
