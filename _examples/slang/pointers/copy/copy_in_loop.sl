// @test: exit_code=0
// @test: stdout=1\n1\n4\n4\n1\n
// Regression: p.copy() inside a loop body or as a branch-expression result
// failed IR validation ("value has no type") — the copy's type was snapshotted
// from an incomplete loop phi. Each copy is freed at its iteration's scope
// exit, so the heap stays balanced.
Point = struct { var x: s64  var y: s64 }

main = () {
    val p = new Point{ 1, 2 }
    for (var i = 0; i < 2; i = i + 1) {
        val q = p.copy()
        print(q.x)           // 1, 1
    }

    val r = new Point{ 3, 4 }
    var j = 0
    while j < 2 {
        val q = r.copy()
        print(q.y)           // 4, 4
        j = j + 1
    }

    val c = true
    val s = if c { p.copy() } else { new Point{ 9, 9 } }
    print(s.x)               // 1
}
