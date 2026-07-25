// @test: exit_code=0
// @test: stdout=Hello, World!\nabc\nab2cd\n5\n0\n
// The + operator concatenates two strings into a fresh heap string
main = () {
    val a = "Hello"
    val b = "World"
    print(a + ", " + b + "!")

    // Chained concatenation of literals folds left-to-right
    print("a" + "b" + "c")

    // Concatenation composes with interpolation
    print("ab" + "${1 + 1}" + "cd")

    // len() of a concatenation frees the temporary
    print(len("ab" + "cde"))
    print(len("" + ""))
}
