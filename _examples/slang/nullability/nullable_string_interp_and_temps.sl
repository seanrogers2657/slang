// @test: exit_code=0
// @test: stdout=g=y2\ng=null\n1\n
// Regression: interpolating a string?-returning call must convert (copy) the
// unwrapped inner BEFORE freeing the temp box — freeing first copied freed
// memory and printed an empty string. And string?-returning call temps must be
// freed when passed as arguments or discarded as statements (they leaked box
// and buffer).
g = (c: bool) -> string? {
    if c {
        return "y${2}"
    }
    return null
}

use = (s: string?) -> s64 {
    if s == null {
        return 0
    }
    return 1
}

main = () {
    print("g=${g(true)}")    // g=y2 — converted before the temp free
    print("g=${g(false)}")   // g=null
    g(true)                  // discarded temp freed, not leaked
    print(use(g(true)))      // 1 — argument temp freed after the call
}
