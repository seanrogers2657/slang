// @test: exit_code=0
// @test: stdout=0\n1\n2\n3\n
// Regression: a bare function call is now valid in the for-loop update clause
// (previously only `identifier = expr` was accepted). Here the update borrows
// the counter mutably and increments it.
Counter = struct { var i: s64 }

bump = (c: &&Counter) {
    c.i = c.i + 1
}

main = () {
    val c = new Counter{ 0 }
    for (var i = 0; c.i < 3; bump(c)) {
        print(c.i)
    }
    print(c.i)
}
