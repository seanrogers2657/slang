// @test: exit_code=0
// @test: stdout=42\n0\n123\n42\n42\n42\n2\n
fn main(): void {
    print(42)
    print(0)
    print(100 + 23)
    print(50 - 8)
    print(6 * 7)
    print(84 / 2)
    print(100 % 7)
}
