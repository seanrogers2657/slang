// @test: exit_code=0
// @test: stdout=11\n99\n
// Regression: methods overloaded by type with the same parameter count must
// emit distinct mangled IR labels. Previously they mangled by arity only and
// collided ("duplicate function name").
Calc = object {
    f = (a: s64) -> s64 { return a + 1 }
    f = (a: string) -> s64 { return 99 }
}

main = () {
    print(Calc.f(10))     // 11
    print(Calc.f("hi"))   // 99
}
