// @test: exit_code=0
// @test: stdout=ok\n
// Regression: struct literals are suppressed in an if/while condition so '{'
// reads as the block start. That suppression must not leak into a nested
// parenthesized or call sub-expression, where '{' is unambiguous. Previously
// `if get_x(Point{1, 2}) > 0 { ... }` failed to parse.
Point = struct {
    val x: s64
    val y: s64
}

get_x = (p: Point) -> s64 {
    return p.x
}

main = () {
    // struct literal as a call argument inside the condition: the '{' here is
    // unambiguous and must still be parsed as a struct literal, not a block.
    if get_x(Point{ 1, 2 }) > 0 {
        print("ok")
    }
}
