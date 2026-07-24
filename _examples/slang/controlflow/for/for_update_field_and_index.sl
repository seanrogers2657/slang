// @test: exit_code=0
// @test: stdout=0\n1\n2\n99\n0\n1\n2\n
// Regression: the for-loop init and update clauses accepted only a bare
// `identifier = expr`. Field assignments (c.i = ...) and index assignments
// (arr[0] = ...) in the update slot tripped "expected ')' after for-loop
// update". They now parse via the same simple-statement path as top-level
// statements.
Counter = struct { var i: s64 }

main = () {
    var c = Counter{ 0 }
    for (var n = 0; c.i < 3; c.i = c.i + 1) {
        print(c.i)
    }
    print(99)

    var arr = [0, 0, 0]
    for (var i = 0; arr[0] < 3; arr[0] = arr[0] + 1) {
        print(arr[0])
    }
}
