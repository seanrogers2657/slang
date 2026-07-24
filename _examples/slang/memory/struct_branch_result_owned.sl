// @test: exit_code=0
// @test: stdout=5\n6\n5\n1\n3\n1\n3\n
// Regression: a struct value flowing out of an if/when-expression branch must
// be owned by the consumer exactly once. A fresh call/new in the taken branch
// leaked (the binding deep-copied the phi result and dropped the branch's
// allocation); a borrowed variable branch must instead be copied so it isn't
// double-freed with its source. Branch bodies now copy borrows and pass fresh
// temps through, and the binding takes ownership without an extra copy. A
// double free would crash, so exit_code=0 guards both directions.
Point = struct { val x: s64  val y: s64 }

make = () -> Point { return Point{ 5, 6 } }

main = () {
    // Fresh call in the taken branch — previously leaked.
    val p = if true { make() } else { make() }
    print(p.x)
    print(p.y)

    val n = 1
    val w = when { n == 1 -> make()  else -> make() }
    print(w.x)

    // Borrowed variable branches — must stay correct and not double-free.
    val a = Point{ 1, 2 }
    val b = Point{ 3, 4 }
    val q = if true { a } else { b }
    val r = if false { a } else { b }
    print(q.x)
    print(r.x)
    print(a.x)
    print(b.x)
}
