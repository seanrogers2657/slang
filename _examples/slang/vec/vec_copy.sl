// @test: exit_code=0
// @test: stdout=1\n1\n2\n
// Copy-on-store: `val b = a` makes an independent vec. Mutating one does not
// affect the other, and both free their own buffer (balanced heap).
main = () {
    var a = vec()
    push(a, 1)
    val b = a            // independent copy
    push(a, 2)           // only a changes
    print(get(b, 0))     // 1
    print(len(b))        // 1
    print(len(a))        // 2
}
