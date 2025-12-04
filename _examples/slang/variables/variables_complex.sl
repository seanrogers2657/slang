// @test: exit_code=0
// @test: stdout=50\n55\n25\n
fn main(): void {
    val a = 5
    val b = 10
    val c = a * b
    print c
    val d = c + a
    print d
    val result = a + b * 2
    print result
}
