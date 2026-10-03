// @test: exit_code=0
// @test: stdout=g12\nn3\nd1\nn4\nn5\nn7\n
// Elvis with a struct result: the result is owned on both edges (a borrowed
// operand is deep-copied, a call-result temp is handed over).
P = struct {
    val x: s64
    var name: string
}

pick = (b: bool, n: s64) -> P? {
    if b { return P{ n, "n${n}" } }
    return null
}

name_of = (p: P?) -> string { return p?.name ?: "none" }

main = () {
    val c = pick(false, 3)
    val g = c ?: P{ 12, "g${12}" }
    print(g.name)

    val a = pick(true, 3)
    val h = a ?: P{ 12, "h${12}" }
    print(h.name)

    val dflt = P{ 1, "d${1}" }
    val i = pick(false, 1) ?: dflt
    print(i.name)

    val j = pick(true, 4) ?: P{ 0, "z" }
    print(j.name)

    print(name_of(pick(true, 5) ?: P{ 0, "z" }))
    pick(true, 6) ?: P{ 0, "z" }

    var k = P{ 0, "z" }
    k = pick(true, 7) ?: k
    print(k.name)
}
