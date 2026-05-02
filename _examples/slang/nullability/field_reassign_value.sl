// @test: exit_code=0
// @test: stdout=7\n42\n
// Reassigning a value-type nullable struct field with a non-null value
// must wrap the new value and produce a readable result.

Box = struct {
    var v: s64?
}

main = () {
    val b = new Box{ 5 }
    b.v = 7
    print(b.v ?: 0)

    b.v = 42
    print(b.v ?: 0)
}
