// @test: exit_code=0
// @test: stdout=60\n
// Test: Heap allocation in function with return value
// Verifies that return values aren't clobbered by deallocation cleanup
Point = struct {
    var x: s64
    var y: s64
    var z: s64
}

allocateAndSum = (a: s64, b: s64, c: s64) -> s64 {
    val p = new Point{ a, b, c }
    return p.x + p.y + p.z
}

main = () {
    val result = allocateAndSum(10, 20, 30)
    print(result)  // 10 + 20 + 30 = 60
}
