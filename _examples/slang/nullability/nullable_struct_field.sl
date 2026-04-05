// @test: exit_code=0
// @test: stdout=7\n0\n
// Nullable struct field access

Box = struct {
    val value: s64?
}

main = () {
    val b = new Box{ 7 }
    val v = b.value ?: 0
    print(v)

    val c = new Box{ null }
    val w = c.value ?: 0
    print(w)
}
