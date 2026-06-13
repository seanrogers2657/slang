// @test: exit_code=0
// @test: stdout=30\n1\n2\n
// Regression: an overloaded method whose parameter is `bool` must bind to its
// own body. Overload-to-body matching compared the type's String() ("boolean")
// against the written name ("bool"), so the bool overload's body was analyzed
// with the wrong overload's parameter types.
Calc = object {
    f = (a: s64, b: s64) -> s64 { return a + b }
    f = (a: bool) -> s64 {
        if a { return 1 }
        return 2
    }
}

main = () {
    print(Calc.f(10, 20))   // 30
    print(Calc.f(true))     // 1
    print(Calc.f(false))    // 2
}
