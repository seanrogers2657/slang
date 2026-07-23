// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot be used as a class field
// Decorator, natural form: hold the wrapped component behind an owned pointer.
// Rejected — *T cannot be a field (it would escape its creating scope).
// (See decorator_ok.sl: embed by value.)
Coffee = class {
    var cost: s64
    price = (self: &Coffee) -> s64 { return self.cost }
}

WithMilk = class {
    var inner: *Coffee
    price = (self: &WithMilk) -> s64 { return self.inner.price() + 1 }
}

main = () {
    val c = new Coffee{ 5 }
    val w = WithMilk{ c }
    print(w.price())
}
