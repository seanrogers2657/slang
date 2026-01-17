// @test: exit_code=0
// @test: stdout=30\n
// Test: Nested heap allocations with proper cleanup
Inner = struct {
    var value: i64
}

Outer = struct {
    var data: *Inner
    var extra: i64
}

createNested = (n: i64) -> i64 {
    val inner = Heap.new(Inner{ n })
    val outer = Heap.new(Outer{ inner, n * 2 })
    return outer.data.value + outer.extra  // n + 2n = 3n
}

main = () {
    val result = createNested(10)
    print(result)  // 10 + 20 = 30
}
