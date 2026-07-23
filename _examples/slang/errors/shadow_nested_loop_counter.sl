// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=shadows a variable declared in an enclosing scope
// Nested loops must use distinct counter names — the inner 'i' would shadow the
// outer 'i' in an enclosing scope, which the IR generator cannot represent.
main = () {
    var sum = 0
    for (var i = 0; i < 3; i = i + 1) {
        for (var i = 0; i < 2; i = i + 1) {   // Error: shadows the outer 'i'
            sum = sum + 1
        }
    }
    print(sum)
}
