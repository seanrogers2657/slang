// @test: exit_code=0
// @test: stdout=11\n22\n11\n7\n
// A top-level binding of struct type is stored in .data like any other, and
// what it holds is the SSA value bound to the name — a pointer to the struct's
// allocation, not the struct layout. Sizing the global from the layout type
// instead makes a field access on it fail IR validation ("FieldPtr argument
// must be a pointer").

Point = struct {
    val x: s64
    var y: s64
}

val ORIGIN = Point{ 11, 22 }

read_x = () -> s64 {
    return ORIGIN.x
}

x_minus = (n: s64) -> s64 {
    return ORIGIN.x - n
}

main = () {
    print(ORIGIN.x)
    print(ORIGIN.y)
    print(read_x())
    print(x_minus(4))
}
