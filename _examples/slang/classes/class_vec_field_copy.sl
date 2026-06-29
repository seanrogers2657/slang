// @test: exit_code=0
// @test: stdout=1\n2\n
// Regression: .copy() of a `new`-allocated class with a vec field deep-copies
// the vec (the class field walk must be handled like a struct's). Growing the
// copy must not change the original, and the heap must stay balanced.
Bag = class {
    var items: vec
    val tag: s64
    count = (self: &Bag) -> s64 { return len(self.items) }
}

main = () {
    val a = new Bag{ vec(), 1 }
    push(a.items, 5)

    val b = a.copy()
    push(b.items, 6)         // grows only the copy

    print(a.count())         // 1  — original unchanged
    print(b.count())         // 2
}
