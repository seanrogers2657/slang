// @test: exit_code=0
// @test: stdout=true\nfalse\n
// Verify that comparison operators return bool type
fn main(): void {
    val a = 5 < 10   // true
    val b = 5 > 10   // false
    print(a)  // true
    print(b)  // false
}
