// @test: exit_code=0
// @test: stdout=1\n2\n
// Reassigning an owned-pointer struct field. Verifies the new value is
// readable and the program does not crash. The old field value should be
// freed by the assignment, but that is not directly observable in stdout.

Inner = struct {
    val v: s64
}

Box = struct {
    var inner: *Inner?
}

main = () {
    val b = new Box{ new Inner{ 1 } }
    print((b.inner ?: new Inner{ 0 }).v)

    b.inner = new Inner{ 2 }
    print((b.inner ?: new Inner{ 0 }).v)
}
