// @test: stdout=30\n10\n100\n10\n7\n2\n
Point = struct {
    var x: s64
    var y: s64
}

// Borrow to read: &T is a read-only reference. The caller keeps ownership.
sum = (p: &Point) -> s64 {
    return p.x + p.y
}

// Borrow to mutate: &&T can change var fields through the reference.
scale = (p: &&Point, factor: s64) {
    p.x = p.x * factor
    p.y = p.y * factor
}

// A factory returns a value (copied to the caller). An owned pointer (*T)
// can never escape its scope, so factories return T, not *T.
make_point = (x: s64, y: s64) -> Point {
    return Point{ x, y }
}

main = () {
    // `new` allocates on the heap. The allocation is freed automatically at the
    // end of this scope — there is no manual free, and ownership never moves.
    val p = new Point{ 10, 20 }
    print(sum(p))       // 30 — *Point auto-borrows to &Point; p is still usable

    // .copy() makes an independent deep copy.
    val q = p.copy()
    print(q.x)          // 10
    scale(p, 10)        // mutate through &&Point (a val binding can still borrow &&T)
    print(p.x)          // 100
    print(q.x)          // 10 — the copy is unaffected by changes to p

    val r = make_point(3, 4)
    print(r.x + r.y)    // 7

    // vec is the growable built-in; like string it is a copyable value type.
    var xs = vec()
    push(xs, 1)
    push(xs, 2)
    print(len(xs))      // 2
}
