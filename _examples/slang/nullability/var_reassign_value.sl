// @test: exit_code=0
// @test: stdout=42\n
// Reassigning a var holding a value-type nullable from another such var
// transfers ownership of the heap slot: only one free at scope exit.

main = () {
    var a: s64? = 1
    var b: s64? = 42
    a = b
    print(a ?: 0)
}
