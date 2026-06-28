// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=cannot be used as a return type
// Builder, natural form: return &&self for fluent chaining. Rejected — a borrow
// is only valid as a parameter, never as a return type.
// (See builder_ok.sl: return a fresh value each step.)
Config = class {
    var width: s64
    set_w = (self: &&Config, w: s64) -> &&Config {
        self.width = w
        return self
    }
}

main = () {
    val c = Config{ 0 }
    print(c.width)
}
