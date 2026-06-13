// @test: exit_code=0
// @test: stdout=99\n1\nhello\nhello\n
// A copyable struct (no owned-pointer fields) bound to a new variable is a
// deep, independent copy: mutating the source's var field must not change the
// copy, and a string field must be duplicated rather than aliased so both
// bindings can free their own buffer at scope exit (balanced heap).
P = struct {
    var x: s64
    val name: string
}

main = () {
    val a = P{ 1, "hello" }
    val b = a       // value copy: b is independent of a
    a.x = 99
    print(a.x)      // 99
    print(b.x)      // 1  (unchanged by the mutation of a)
    print(a.name)   // hello
    print(b.name)   // hello (independent buffer)
}
