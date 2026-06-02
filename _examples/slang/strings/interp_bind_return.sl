// @test: exit_code=0
// @test: stdout=greeting: Hi Sam\nlabel #1\n
// Interpolated strings bound to variables and returned from functions, with a
// string-returning call used inside another interpolation (heap stays balanced).
greet = (who: string) -> string {
    return "Hi ${who}"
}

make_label = (n: s64) -> string {
    return "#${n}"
}

main = () {
    val g = "greeting: ${greet("Sam")}"
    print(g)
    val l = make_label(1)
    print("label ${l}")
}
