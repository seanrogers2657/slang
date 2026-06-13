// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=moved value
// Regression: calling a method whose receiver is an owned pointer (self: *T)
// consumes the receiver, just like passing *T to a free function. Calling it
// twice must be a use-after-move error; previously the move went unrecorded.
Box = class {
    var v: s64

    consume = (self: *Box) -> s64 {
        return self.v
    }
}

main = () {
    val b = new Box{ 5 }
    print(b.consume())   // moves b
    print(b.consume())   // Error: use of moved value 'b'
}
