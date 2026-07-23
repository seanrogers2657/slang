// @test: exit_code=0
// @test: stdout=1\n2\n
// Regression: a string?-returning call used directly in a null comparison
// (if mk() != null) produced a temp that no site freed — the comparison
// operand is a consumer position too.
mk = (c: bool) -> string? {
    if c {
        return "v${1}"
    }
    return null
}

main = () {
    if mk(true) != null {
        print(1)
    }
    if mk(false) == null {
        print(2)
    }
}
