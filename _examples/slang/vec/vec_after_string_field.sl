// @test: exit_code=0
// @test: stdout=1\n2\na=1\n
// Regression: a vec field placed after a string field lives at IR offset 16
// (strings are 16 bytes in the IR layout), not semantic offset 8. Deep-copy
// (copy-on-store `var b = a`) and deep-free must both use the IR layout or the
// vec header is copied/freed from the wrong slot — this used to segfault.
S = struct {
    var name: string
    var items: vec
}

main = () {
    val n = 1
    var a = S{ "a=${n}", vec() }
    push(a.items, 10)

    var b = a               // copy-on-store: b owns independent copies
    push(b.items, 20)

    print(len(a.items))     // 1 — a unaffected by b's push
    print(len(b.items))     // 2
    print(b.name)           // a=1 — b's string copied at the right offset
}
