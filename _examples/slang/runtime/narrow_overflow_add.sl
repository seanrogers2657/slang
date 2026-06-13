// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: addition
// Regression: arithmetic on narrow integer types must detect overflow. The
// 64-bit flag checks only catch overflow at the 64-bit boundary, so s8 100+100
// silently produced 200 instead of trapping.
main = () {
    val a: s8 = 100
    val b: s8 = 100
    val c: s8 = a + b
    print(c)
}
