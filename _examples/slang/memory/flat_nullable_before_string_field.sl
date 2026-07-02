// @test: exit_code=0
// @test: stdout=n=4\n
// Regression: the deep-free walk must use the IR field layout, not the
// semantic one. A flat nullable (s64?) occupies 16 bytes in the IR layout, so
// a string field placed after it lives at a different offset than the semantic
// 8-bytes-per-field layout claims. Freeing at the wrong offset leaked the real
// string buffer (and freed a garbage word) — the runtime aborted on the
// unbalanced heap.
S = struct {
    var a: s64?
    var name: string
}

main = () {
    val n = 4
    var s = S{ 9, "n=${n}" }
    print(s.name)
}
