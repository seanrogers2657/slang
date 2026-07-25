// @test: exit_code=0
// @test: stdout=abab\nabab\nxyxy\nllowo\nbcd\n[xy]\n[el]\ntrue\nab\nabcd\nab\nwx\nab\nel\nZ\n45\n
// Every substr/concat/chr result is a fresh heap allocation with no binding to
// free it. This exercises the positions where such a temporary can appear —
// nested, aliased, returned, stored, branched, and passed — and relies on the
// runtime's balanced-heap assertion at exit to catch a leak or double free.

mk = () -> string {
    return "x" + "y"
}

join = (a: string, b: string) -> string {
    return a + b
}

head = (s: string, n: s64) -> string {
    return substr(s, 0, n)
}

show = (s: string) {
    print(s)
}

P = struct {
    var name: string
}

main = () {
    // Same binding on both sides — neither side may be freed
    val s = "ab"
    print(s + s)

    // Reassignment through self-reference frees the old buffer
    var acc = "ab"
    acc = acc + acc
    print(acc)

    // Two independent owned temporaries in one expression
    print(mk() + mk())

    // Temporary consumed by another builtin
    print(substr("hello" + "world", 2, 7))
    print(substr(head("abcdef", 5), 1, 4))

    // Temporaries inside interpolation
    val a = "x"
    print("[${a + "y"}]")
    print("[${substr("hello", 1, 3)}]")

    // Temporary consumed by a comparison
    print(("a" + "b") == "ab")

    // Returned across a call boundary
    print(join("a", "b"))
    print(head("abcdef", 4) + "")

    // Stored into a struct field, both at construction and by assignment
    val p = P{ "a" + "b" }
    print(p.name)
    var q = P{ "zz" }
    q.name = substr("wxyz", 0, 2)
    print(q.name)

    // Yielded from branch expressions
    val n = 5
    val fromIf = if n > 0 { "a" + "b" } else { substr("qrs", 0, 1) }
    print(fromIf)
    val fromWhen = when { n > 3 -> substr("hello", 1, 3) else -> "no" }
    print(fromWhen)

    // Passed directly as an argument
    show(chr(90))

    // Many short-lived temporaries in a loop
    val src = "abcdefghij"
    var total = 0
    for (var i = 0; i < len(src); i = i + 1) {
        total = total + len(substr(src, 0, i))
    }
    print(total)
}
