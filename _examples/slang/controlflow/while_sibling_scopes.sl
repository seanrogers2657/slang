// @test: exit_code=0
// @test: stdout=3\n7\n
// Regression: a while body must have its own lexical scope (like if and for
// bodies). Declaring `val step` in two sibling while loops used to fail with
// "variable 'step' is already declared in this scope".
main = () {
    var a = 0
    var i = 0
    while i < 3 {
        val step = 1
        a = a + step
        i = i + 1
    }
    print(a)      // 3

    var j = 0
    while j < 2 {
        val step = 2      // same name, sibling loop — its own scope
        a = a + step
        j = j + 1
    }
    print(a)      // 7
}
