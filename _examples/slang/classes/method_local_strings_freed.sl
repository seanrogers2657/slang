// @test: exit_code=0
// @test: stdout=2\n2\nnote7\n
// Regression: generateMethod never initialized the owned-variable scope
// stack, so heap-owning method locals (interpolated strings, elvis-unwrap
// copies of string? fields) were never tracked and leaked one buffer per
// call. Free functions were unaffected — only methods.
G = class {
    var n: s64
    val note: string?
    f = (self: &G) -> s64 {
        val s = "x${self.n}"       // inline heap local — freed at return
        return len(s)
    }
    describe = (self: &G) -> string {
        val d = self.note ?: "-"   // elvis-unwrap copy — freed or returned
        return d
    }
}

main = () {
    val g = new G{ 1, "note${7}" }
    print(g.f())          // 2
    print(g.f())          // 2 — second call must not leak either
    print(g.describe())   // note7
}
