// @test: exit_code=0
// @test: stdout=6\n60\n
// Regression: a class with a nested aggregate VALUE field (struct or class).
// The field is embedded inline (copied), not stored as a pointer, and the
// temporary is freed — the heap stays balanced.
Inner = struct { var cost: s64 }

Boxed = class {
    var inner: Inner
    var tag: s64
    total = (self: &Boxed) -> s64 { return self.inner.cost + self.tag }
}

main = () {
    val b = Boxed{ Inner{ 5 }, 1 }
    print(b.total())                       // 6
    val h = new Boxed{ Inner{ 50 }, 10 }
    print(h.total())                       // 60
}
