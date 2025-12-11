// @test: exit_code=0
// @test: stdout=true\n
// Tests complex boolean expressions with mixed operators
fn main(): void {
    val x = 5
    val y = 10
    val cond1 = x < y      // true
    val cond2 = y > 0      // true
    val result = cond1 && cond2 || false
    print(result)  // true
}
