// @test: exit_code=0
// @test: stdout=42\n
Wrapper = struct {
    val value: s64
}

main = () {
    val p = Heap.new(Wrapper{ 42 })
    print(p.value)
}
