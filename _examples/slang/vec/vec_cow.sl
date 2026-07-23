// @test: exit_code=0
// @test: stdout=99\n1\n4\n3\n4\n3\n
// Copy-on-write: `val b = a` shares the buffer in O(1) (a refcount bump, no
// element copy). The deep copy happens lazily, only when a shared vec is
// mutated (set/push), so the two bindings stay independent — value semantics
// without copying on every bind. The heap stays balanced.
main = () {
    var a = vec()
    push(a, 1)
    push(a, 2)
    push(a, 3)
    val b = a            // shares a's buffer (O(1))

    set(a, 0, 99)        // mutating a uniquifies it; b is unaffected
    push(a, 4)

    print(get(a, 0))     // 99
    print(get(b, 0))     // 1
    print(len(a))        // 4
    print(len(b))        // 3

    val c = b
    push(b, 7)           // mutating b uniquifies it; c is unaffected
    print(len(b))        // 4
    print(len(c))        // 3
}
