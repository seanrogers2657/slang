// @test: exit_code=0
// @test: stdout=10\nfalse\n20\ntrue\n
// Test struct with nullable field
Person = struct {
    val age: s64
    var score: s64?
}

main = () {
    val p1 = Person{ 10, null }
    print(p1.age)           // 10
    val s1 = p1.score
    print(s1 != null)       // false

    val p2 = Person{ 20, 100 }
    print(p2.age)           // 20
    val s2 = p2.score
    print(s2 != null)       // true
}
