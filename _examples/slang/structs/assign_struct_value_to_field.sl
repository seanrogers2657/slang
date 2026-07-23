// @test: exit_code=0
// @test: stdout=7\n99\na1\nb2\nc3\n
// Regression: assigning a struct VALUE to an embedded struct field must copy
// the value into the inline slot (MemCopy), not store a pointer. Storing a
// pointer read back garbage and leaked the RHS. Heap-owning sub-fields must be
// deep-copied (independent) and the old field contents freed.
Inner = struct { var name: string }
Plain = struct { var a: s64 }
Outer = struct { var inner: Inner  var p: Plain }

main = () {
    var o = Outer{ Inner{ "a${1}" }, Plain{ 7 } }
    print(o.p.a)              // 7

    o.p = Plain{ 99 }
    print(o.p.a)              // 99 — old Plain freed, new copied in

    print(o.inner.name)       // a1
    o.inner = Inner{ "b${2}" }
    print(o.inner.name)       // b2 — old Inner's string freed, new copied

    var src = Inner{ "c${3}" }
    o.inner = src
    src.name = "x${9}"        // mutate the source
    print(o.inner.name)       // c3 — field owns an independent copy
}
