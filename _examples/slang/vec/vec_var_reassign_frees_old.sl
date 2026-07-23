// @test: exit_code=0
// @test: stdout=3\n2\n2\n49\n
// Regression: reassigning one vec variable from another (s = t) must free the
// vec s previously owned and give s an independent copy of t (copy-on-store,
// like string). The old code flagged the source vec as moved-away, so its
// buffer was never freed at scope exit — a heap leak. It must also stay a real
// copy: mutating one side must not affect the other, and a reassignment loop
// must free each replaced vec.
main = () {
    var s = vec()
    push(s, 1)
    var t = vec()
    push(t, 2)
    push(t, 3)

    s = t              // s frees its old buffer and takes a copy of t
    push(s, 9)         // grows only s's copy
    print(len(s))      // 3
    print(len(t))      // 2 — t unaffected
    print(get(t, 0))   // 2 — t's contents intact

    // Reassignment in a loop must free each replaced vec (heap stays balanced).
    var acc = vec()
    var i = 0
    while i < 50 {
        var tmp = vec()
        push(tmp, i)
        acc = tmp
        i = i + 1
    }
    print(get(acc, 0))   // 49
}
