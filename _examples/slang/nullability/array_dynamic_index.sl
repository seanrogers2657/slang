// @test: exit_code=0
// @test: stdout=true\nfalse\n
// Test array with dynamic index access
main = () {
    val x: s64? = 42
    val y: s64? = null
    val arr = [x, y]

    var i = 0
    val a = arr[i]
    print(a != null)  // true

    i = 1
    val b = arr[i]
    print(b != null)  // false
}
