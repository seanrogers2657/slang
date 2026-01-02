// @test: exit_code=0
// @test: stdout=true\nfalse\n
// Test array with nullable elements (simplified to avoid large stack offset)
main = () {
    val x: i64? = 42
    val y: i64? = null
    val arr = [x, y]

    val a = arr[0]
    print(a != null)  // true

    val b = arr[1]
    print(b != null)  // false
}
