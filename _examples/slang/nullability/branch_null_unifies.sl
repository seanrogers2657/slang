// @test: exit_code=0
// @test: stdout=n5\n-\n42\nnull\ns1\ng9\n-\none1\nn2\n3\n
// An if/when branch yielding null unifies with a T or T? branch to T?. The
// null branch emits a typed null, which is also safe as a returned value.
P = struct {
    val x: s64
    var name: string
}
pick = (b: bool, n: s64) -> P? {
    if b { return P{ n, "n${n}" } }
    return null
}
f = (k: s64) -> P? {
    return when {
        k == 0 -> null
        k == 1 -> P{ 1, "one${1}" }
        else -> pick(true, k)
    }
}
main = () {
    val a = when { true -> pick(true, 5) else -> null }
    print(a?.name ?: "-")
    val b = if false { pick(true, 1) } else { null }
    print(b?.name ?: "-")
    val c = if true { 42 } else { null }
    print("${c}")
    val d = when { false -> 1 else -> null }
    print("${d}")
    val e = if true { "s${1}" } else { null }
    print("${e}")
    val g = if false { null } else { P{ 9, "g${9}" } }
    print(g?.name ?: "-")
    print(f(0)?.name ?: "-")
    print(f(1)?.name ?: "-")
    print(f(2)?.name ?: "-")
    val x: s64? = 3
    val h = if true { x } else { 4 }
    print("${h}")
}
