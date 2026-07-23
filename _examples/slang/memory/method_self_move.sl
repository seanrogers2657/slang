// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains='self' cannot take ownership
// A method receiver cannot take ownership (self: *T); consuming methods are
// not allowed. Use self: &T to read or self: &&T to mutate.
Box = class {
    var v: s64

    consume = (self: *Box) -> s64 {  // Error: self cannot take ownership
        return self.v
    }
}

main = () {
    val b = new Box{ 5 }
    print(b.consume())
    print(b.consume())
}
