// @test: exit_code=0
// @test: stdout=99\n42\n
// Guard: mutation through a mutable && borrow must remain allowed at any depth
// (the immutable-&T nested-mutation fix must not over-reject &&T). Covers both
// a free function taking &&Outer and a class method with &&self.
Inner = struct { var v: s64 }
Outer = struct { var i: Inner }

mutate = (o: &&Outer) {
    o.i.v = 99
}

Box = class {
    var i: Inner
    set = (self: &&Box) {
        self.i.v = 42
    }
}

main = () {
    val o = new Outer{ Inner{ 1 } }
    mutate(o)
    print(o.i.v)

    val b = new Box{ Inner{ 0 } }
    b.set()
    print(b.i.v)
}
