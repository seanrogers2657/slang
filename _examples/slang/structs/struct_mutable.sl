// @test: exit_code=0
// @test: stdout=10\n25\n
// Demonstrates mutable struct fields with var keyword
Point = struct {
    val x: s64
    var y: s64
}

main = () {
    val p = Point{ 10, 20 }
    p.y = 25
    print(p.x)
    print(p.y)
}
