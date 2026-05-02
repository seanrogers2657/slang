// @test: exit_code=0
// @test: stdout=42\n
// Returning a local value-type nullable transfers ownership of the heap
// slot to the caller — the function must not free it on the way out, and
// the caller's binding owns it until its own scope ends.

make_value = () -> s64? {
    val x: s64? = 42
    return x
}

main = () {
    val r = make_value()
    print(r ?: 0)
}
