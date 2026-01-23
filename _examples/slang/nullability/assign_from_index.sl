// @test: exit_code=0
// @test: stdout=false\ntrue\n
// Test assigning nullable values from array index expressions
main = () {
    val x: s64? = 42
    val y: s64? = null
    val arr = [x, y]

    // Assign from index to mutable variable - exercises TypedIndexExpr in generateNullableAssign
    var a: s64? = 100
    a = arr[1]  // assign null from index
    print(a != null)  // false

    a = arr[0]  // assign non-null from index
    print(a != null)  // true
}
