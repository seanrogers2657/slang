// @test: stdout=hi\nq1\nr2\n
// Returning an embedded field, a parameter, or an array element by value
// gives the caller an independent copy rather than an alias of storage
// that is freed elsewhere.
Inner = struct {
    var s: string
}

Outer = struct {
    var inner: Inner
}

field_of = (o: Outer) -> Inner {
    return o.inner
}

echo = (p: Inner) -> Inner {
    return p
}

first = (a: Inner[]) -> Inner {
    return a[0]
}

main = () {
    val o = Outer{ Inner{ "hi" } }
    val x = field_of(o)
    print(x.s)

    val a = Inner{ "q${1}" }
    val y = echo(a)
    print(y.s)

    val arr = [Inner{ "r${2}" }]
    val z = first(arr)
    print(z.s)
}
