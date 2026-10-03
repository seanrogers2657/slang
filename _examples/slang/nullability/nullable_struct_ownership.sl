// @test: exit_code=0
// @test: stdout=a1\nn2\nnone\np4\nn2\nf6\na1\nn8\n9\nn10\nn11\nn10\nn0\nnone\nn2\n
// A nullable struct (P?) binding owns its struct: it is freed at scope exit,
// deep-copied when bound from another binding or a field, and freed when
// reassigned, discarded, or passed as a call-result temporary.
P = struct {
    val x: s64
    var name: string
}
Box = struct {
    var p: P?
}

pick = (b: bool, n: s64) -> P? {
    if b { return P{ n, "n${n}" } }
    return null
}

pass = (p: P?) -> P? { return p }

name_of = (p: P?) -> string { return p?.name ?: "none" }

main = () {
    val a: P? = P{ 1, "a${1}" }
    val b = pick(true, 2)
    val c = pick(false, 3)
    print(name_of(a))
    print(name_of(b))
    print(name_of(c))

    // bind from an existing P and from an existing P?
    val plain = P{ 4, "p${4}" }
    val d: P? = plain
    val e: P? = b
    print(name_of(d))
    print(name_of(e))

    // reassign
    var f: P? = null
    f = a
    f = pick(true, 5)
    f = null
    f = P{ 6, "f${6}" }
    print(name_of(f))

    // through a function and discarded
    print(name_of(pass(a)))
    pick(true, 7)
    print(pick(true, 8)?.name ?: "x")
    print(pick(true, 9)?.x ?: 0)

    // P? inside a struct, copied
    val bx = Box{ pick(true, 10) }
    var by = bx
    by.p = pick(true, 11)
    print(name_of(bx.p))
    print(name_of(by.p))
    val inner: P? = bx.p
    print(name_of(inner))

    // in a loop
    for (var i = 0; i < 3; i = i + 1) {
        val t = pick(i % 2 == 0, i)
        print(name_of(t))
    }

}
