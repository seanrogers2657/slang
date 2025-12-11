// @test: exit_code=0
// @test: stdout=true\nfalse\nfalse\nfalse\n
// Tests logical AND operator truth table
fn main(): void {
    print(true && true)   // true
    print(true && false)  // false
    print(false && true)  // false
    print(false && false) // false
}
