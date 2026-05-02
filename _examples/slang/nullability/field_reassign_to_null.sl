// @test: exit_code=0
// @test: stdout=true\nfalse\ntrue\n
// Reassigning a value-type nullable field to null and back to a value.

Box = struct {
    var v: s64?
}

main = () {
    val b = new Box{ 1 }
    print(b.v != null)

    b.v = null
    print(b.v != null)

    b.v = 99
    print(b.v != null)
}
