// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=duplicate method signature
// Error: duplicate method signature (same name and parameter types)

Printer = class {
    var count: s64

    // Overload 1: accepts i64
    print = (self: &&Printer, x: s64) {
        self.count = self.count + 1
    }

    // Overload 2: same signature - should be rejected
    print = (self: &&Printer, y: s64) {
        self.count = self.count + 2
    }
}

main = () {
    val p = new Printer{ 0 }
    p.print(42)
}
