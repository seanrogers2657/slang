// @test: exit_code=0
// @test: stdout=60\n
// Test: Heap allocation in function with return value
// Verifies that return values aren't clobbered by deallocation cleanup
Point = struct {
    var x: i64
    var y: i64
    var z: i64
}

allocateAndSum = (a: i64, b: i64, c: i64) -> i64 {
    val p = Heap.new(Point{ a, b, c })
    return p.x + p.y + p.z
}

main = () {
    val result = allocateAndSum(10, 20, 30)
    print(result)  // 10 + 20 + 30 = 60
}
