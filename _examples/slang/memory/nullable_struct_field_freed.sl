// @test: exit_code=0
// @test: stdout=false\ntrue\n2\n
// Regression: a nullable struct-typed field (Inner?) holds a pointer to a
// separately heap-allocated struct, but it fell through every case in the
// aggregate-free walk (nullableValueInner excludes structs, and it is not an
// owned pointer), leaking the pointee. It is now null-checked and recursively
// freed like a nullable owned pointer. Reassigning frees the old one exactly
// once (a double free would crash, so exit_code=0 guards both directions).
Inner = struct { val v: s64 }
Outer = struct { var inner: Inner? }

main = () {
    val a = Outer{ Inner{ 1 } }
    print(a.inner == null)      // false
    val b = Outer{ null }
    print(b.inner == null)      // true

    var c = Outer{ Inner{ 1 } }
    c = Outer{ Inner{ 2 } }     // frees the first Inner exactly once
    print(c.inner?.v ?: -1)     // 2
}
