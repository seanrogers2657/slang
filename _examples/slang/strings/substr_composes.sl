// @test: exit_code=0
// @test: stdout=ell\nHeo, World!\ntrue\n4\n
// substr composes with concatenation and with itself; every intermediate is a
// fresh heap string that must be freed once consumed.
main = () {
    val s = "Hello, World!"

    // substr of a substr
    print(substr(substr(s, 0, 5), 1, 4))

    // Cutting a range out of the middle by concatenating two slices
    print(substr(s, 0, 2) + substr(s, 4, len(s)))

    // Slices compare by value, not identity
    print(substr(s, 0, 5) == "Hello")

    // len() of a fresh substr temporary
    print(len(substr(s, 1, 5)))
}
