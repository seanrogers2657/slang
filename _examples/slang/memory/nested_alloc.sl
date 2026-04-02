// @test: exit_code=0
// @test: stdout=30\n
// Test: Nested heap allocations with proper cleanup
Inner = struct {
    var value: s64
}

Outer = struct {
    var data: *Inner
    var extra: s64
}

createNested = (n: s64) -> s64 {
    val inner = new Inner{ n }
    val outer = new Outer{ inner, n * 2 }
    return outer.data.value + outer.extra  // n + 2n = 3n
}

main = () {
    val result = createNested(10)
    print(result)  // 10 + 20 = 30
}
