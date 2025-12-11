// @test: exit_code=0
// @test: stdout=true\n
// Verify short-circuit: right side should NOT be evaluated when left is true
fn side_effect(): bool {
    print(999)  // This should NOT print if short-circuit works
    return false
}

fn main(): void {
    val result = true || side_effect()
    print(result)  // prints true
}
