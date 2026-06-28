// @test: exit_code=0
// @test: stdout=1120\n
// Builder pattern. Fluent chaining works by returning a fresh Config VALUE from
// each step (you cannot return &&self — a borrow can't be a return type).
Config = class {
    var width: s64
    var height: s64

    set_w = (self: &Config, w: s64) -> Config { return Config{ w, self.height } }
    set_h = (self: &Config, h: s64) -> Config { return Config{ self.width, h } }
    total = (self: &Config) -> s64 { return self.width + self.height }
}

main = () {
    val c = Config{ 0, 0 }
    val d = c.set_w(640).set_h(480)
    print(d.total())
}
