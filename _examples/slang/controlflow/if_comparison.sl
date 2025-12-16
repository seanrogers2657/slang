// @test: exit_code=0
// @test: stdout=equal\n
// Tests if with various comparison operators
fn main(): void {
    val a = 5
    val b = 5
    if a == b {
        print("equal")
    } else {
        print("not equal")
    }
}
