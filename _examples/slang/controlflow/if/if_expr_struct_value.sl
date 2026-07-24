// @test: exit_code=0
// @test: stdout=1\n2\n3\n4\n
// Regression: an if-expression whose branches yield a struct value must
// compile. The result phi was typed with the bare value type (Point) while the
// branch operands are pointers to the struct allocation (*Point), so IR
// validation rejected it. The phi now uses the SSA type (pointer for
// aggregates), matching the when-expression path.
Point = struct { val x: s64  val y: s64 }

main = () {
    val a = Point{ 1, 2 }
    val b = Point{ 3, 4 }
    val p = if true { a } else { b }
    val q = if false { a } else { b }
    print(p.x)
    print(p.y)
    print(q.x)
    print(q.y)
}
