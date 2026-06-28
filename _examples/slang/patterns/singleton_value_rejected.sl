// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=expected &&Registry
// Shared mutable state, natural form: pass a plain stack value to a function
// that takes &&T. Rejected — a plain value does not auto-borrow; only an owned
// pointer (from `new`) auto-borrows into &T/&&T.
// (See singleton_ok.sl: allocate the instance with `new`.)
Registry = struct { var count: s64 }

bump = (r: &&Registry) { r.count = r.count + 1 }

main = () {
    var g = Registry{ 0 }
    bump(g)
    print(g.count)
}
