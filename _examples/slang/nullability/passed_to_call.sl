// @test: exit_code=0
// @test: stdout=7\n7\n7\n
// Passing a value-type nullable to a function uses borrow semantics: the
// callee may read the parameter but does not free its heap slot, so the
// caller can reuse the variable across multiple calls.

print_it = (x: s64?) {
    print(x ?: 0)
}

main = () {
    val a: s64? = 7
    print_it(a)
    print_it(a)
    print(a ?: 0)
}
