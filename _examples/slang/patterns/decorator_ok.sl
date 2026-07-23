// @test: exit_code=0
// @test: stdout=6\n
// Decorator pattern. The wrapper embeds the wrapped component BY VALUE (a *T
// field is illegal under scope-frees-it) and forwards to it through a method.
Coffee = class {
    var cost: s64
    price = (self: &Coffee) -> s64 { return self.cost }
}

WithMilk = class {
    var inner: Coffee
    price = (self: &WithMilk) -> s64 { return self.inner.price() + 1 }
}

main = () {
    val w = WithMilk{ Coffee{ 5 } }
    print(w.price())
}
