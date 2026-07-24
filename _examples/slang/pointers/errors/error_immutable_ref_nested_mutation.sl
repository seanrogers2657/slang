// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot assign through immutable reference
// Regression: an immutable &T borrow grants read-only access to the entire
// reachable object graph. Mutation through it was only blocked at the first
// field level (p.x = ...); a nested target (o.i.v = ...) slipped through
// because the intermediate field access analyzed to a bare struct value and
// lost the &T wrapper. The mutability check now walks to the root binding.
Inner = struct { var v: s64 }
Outer = struct { var i: Inner }

mutate = (o: &Outer) {
    o.i.v = 99   // Error: cannot mutate through an immutable &Outer borrow
}

main = () {
    val o = new Outer{ Inner{ 1 } }
    mutate(o)
    print(o.i.v)
}
