// @test: exit_code=0
// @test: stdout=88\n
fn main(): void {
    val a: i64 = 42
    val b: i64 = 2
    val c = a + b  // c = 44
    val d = c * b  // d = 88
    print(d)
}
