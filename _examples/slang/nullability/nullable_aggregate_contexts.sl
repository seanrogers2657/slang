// @test: exit_code=0
// @test: stdout=n3\nn3\nn3\nn5\nin1\nn6\nn6\nl7\nC:k1\nn9\nn0\nn2\n
// Nullable aggregates in arrays, branches, nested fields, returns, classes,
// globals, and loops. ?. through an embedded struct field copies it out of
// the inline storage, and a field after a string uses the IR layout offset.
P = struct {
    val x: s64
    var name: string
}
W = struct {
    val inner: P
    var opt: P?
}
C = class {
    var label: string
    show = (self: &C) -> string { return "C:${self.label}" }
}
pick = (b: bool, n: s64) -> P? {
    if b { return P{ n, "n${n}" } }
    return null
}
first = (w: &W) -> P? { return w.opt }
local_ret = () -> P? {
    val l: P? = P{ 7, "l${7}" }
    return l
}
var gp: P? = null

main = () {
    // arrays of P?
    var arr = [pick(true, 1), pick(false, 2)]
    arr[1] = pick(true, 3)
    arr[0] = null
    print(arr[1]?.name ?: "-")
    val copied = arr
    print(copied[1]?.name ?: "-")

    // if / when producing P?
    val i1 = if arr[1] == null { pick(true, 4) } else { arr[1] }
    print(i1?.name ?: "-")
    val w1 = when { true -> pick(true, 5) else -> pick(false, 0) }
    print(w1?.name ?: "-")

    // nested
    val w = W{ P{ 1, "in${1}" }, pick(true, 6) }
    val ow: W? = w
    print(ow?.inner?.name ?: "-")
    print(ow?.opt?.name ?: "-")
    var w2 = w
    w2.opt = w.opt
    w2.opt = null
    print(w.opt?.name ?: "-")

    // returns
    print(local_ret()?.name ?: "-")

    // class nullable
    val c: C? = C{ "k${1}" }
    print(c?.show() ?: "-")

    // global
    gp = pick(true, 8)
    gp = pick(true, 9)
    print(gp?.name ?: "-")

    // loop reassign
    var cur: P? = null
    for (var i = 0; i < 4; i = i + 1) {
        cur = pick(i % 2 == 0, i)
        if cur == null { continue }
        print(cur?.name ?: "-")
    }
}
