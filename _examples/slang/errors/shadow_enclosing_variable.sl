// @test: expect_error=true
// @test: error_stage=semantic
// @test: error_contains=shadows a variable declared in an enclosing scope
// A variable declared in a nested block may not reuse the name of a variable in
// an enclosing scope. The IR generator keys SSA variables by bare name and
// cannot distinguish the inner binding from the outer, silently collapsing them
// (a wrong value, or a heap leak/crash for heap-owning types). Rejecting the
// shadow keeps this safe until scope-qualified SSA names exist.
main = () {
    val x = 5
    var i = 0
    while i < 3 {
        val x = 100   // Error: shadows the enclosing 'x'
        print(x)
        i = i + 1
    }
    print(x)
}
