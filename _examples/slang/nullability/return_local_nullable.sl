// @test: exit_code=0
// @test: stdout=42\n
// A value-type nullable is returned by value (like any value): its heap slot
// becomes the caller's, so the function must not free it on the way out, and
// the caller's binding holds it until that binding's scope ends.

make_value = () -> s64? {
    val x: s64? = 42
    return x
}

main = () {
    val r = make_value()
    print(r ?: 0)
}
