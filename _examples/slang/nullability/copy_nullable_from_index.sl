// @test: exit_code=0
// @test: stdout=true\nfalse\n
// Test copying nullable values from array index expressions to another array
main = () {
    val x: s64? = 42
    val y: s64? = null
    val src = [x, y]

    // Copy from index to another array - this exercises TypedIndexExpr in generateArrayVarDecl
    val dest = [src[0], src[1]]
    val d0 = dest[0]
    val d1 = dest[1]
    print(d0 != null)  // true
    print(d1 != null)  // false
}
