// @test: stdout=5\nfalse\n5\ndeep null\nnull\nnull\nnull null\ndflt\ntmp\ndeep\ndeep\ndeep\n
// Safe navigation to a field that is itself nullable yields that field's
// value (T?, not T??): it must not be wrapped a second time, which read
// back the inner presence flag (or crashed for a string? field).
P = struct {
    val b: s64
    val w: s64?
}
Inner = struct {
    val w: string?
    val n: s64?
}
Outer = struct {
    val inner: Inner?
}
make = (s: string?) -> Outer? {
    return Outer{ Inner{ s, 4 } }
}
main = () {
    val q: P? = P{ 1, 5 }
    print("${q?.w}")
    val r = q?.w
    print(r == null)
    print(r ?: 99)

    val a: Outer? = Outer{ Inner{ "deep", null } }
    print("${a?.inner?.w} ${a?.inner?.n}")
    val b: Outer? = Outer{ null }
    print("${b?.inner?.w}")
    val c: Outer? = null
    print("${c?.inner?.w}")
    val d: Inner? = Inner{ null, null }
    print("${d?.w} ${d?.n}")
    val e = d?.w ?: "dflt"
    print(e)
    print("${make("tmp")?.inner?.w}")
    for (var i = 0; i < 3; i = i + 1) {
        val s = a?.inner?.w
        print(s ?: "x")
    }
}
