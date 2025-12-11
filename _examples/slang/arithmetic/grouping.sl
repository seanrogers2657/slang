// @test: exit_code=0
// @test: stdout=20\n14\n21\n
fn main(): void {
    // (2 + 3) * 4 = 20 (grouping overrides precedence)
    print((2 + 3) * 4)

    // 2 + 3 * 4 = 14 (normal precedence)
    print(2 + 3 * 4)

    // (1 + 2) * (3 + 4) = 3 * 7 = 21
    print((1 + 2) * (3 + 4))
}
