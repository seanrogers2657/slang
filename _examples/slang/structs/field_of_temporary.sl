// @test: stdout=4\nx1\nx1\ny2\ny2\n7\nz3\nx1!\nx1\nb2\n104\na1\na1!\n
// Reading a field or element straight off a call result or literal copies
// the value out and frees the temporary, so nothing leaks and nothing is freed twice.
Pair = struct {
    var s: string
    var n: s64
}

Inner = struct {
    var s: string
}

Outer = struct {
    var inner: Inner
    var k: s64
}

make_pair = () -> Pair {
    return Pair{ "x${1}", 4 }
}

make_inners = () -> Inner[] {
    return [Inner{ "a${1}" }, Inner{ "b${2}" }]
}

make_word = () -> string {
    return "h${9}"
}

make_outer = () -> Outer {
    return Outer{ Inner{ "y${2}" }, 7 }
}

main = () {
    print(make_pair().n)
    print(make_pair().s)
    val t = make_pair().s
    print(t)

    print(make_outer().inner.s)
    val q = make_outer().inner
    print(q.s)
    print(make_outer().k)

    print(Pair{ "z${3}", 1 }.s)
    print(make_pair().s + "!")
    print("${make_pair().s}")

    print(make_inners()[1].s)
    print(make_word()[0])
    val e = make_inners()[0]
    print(e.s)
    print(make_inners()[0].s + "!")
}
