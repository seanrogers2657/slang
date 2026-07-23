// @test: expect_error=true
// @test: error_stage=parser
// @test: error_contains=expected type name
// Strategy, natural form: pass an algorithm as a function value. Rejected —
// Slang has no function types / first-class functions.
// (See strategy_ok.sl: dispatch on an integer tag.)
apply = (op: (s64, s64) -> s64, a: s64, b: s64) -> s64 {
    return op(a, b)
}

main = () {
    print(apply(add, 3, 4))
}
