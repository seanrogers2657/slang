// @test: exit_code=0
// @test: stdout=9\n5\n6\n
// Regression: method calls borrow their arguments like free functions, but
// generateMethodCall had none of the argument-temp freeing — an elvis result,
// a boxed string?-returning call, or an interpolated string passed to a
// method leaked per call.
C = class {
    val base: s64
    use = (self: &C, s: string) -> s64 {
        return self.base + len(s)
    }
    use_opt = (self: &C, s: string?) -> s64 {
        return self.base + (if s == null { 0 } else { 1 })
    }
}

mk = () -> string? {
    return "abc${1}"
}

main = () {
    val c = C{ 4 }
    val m: string? = "left${5}"
    print(c.use(m ?: "dd"))   // 4 + len("left5") = 9 — elvis temp freed
    print(c.use_opt(mk()))    // 4 + 1 = 5 — boxed call temp freed
    print(c.use("v${7}"))     // 4 + len("v7") = 6 — interp temp freed
}
