// @test: exit_code=50
// Tests when expression used directly in return
fn getValue(x: i64): i64 {
    return when {
        x > 100 -> 100
        x > 10 -> 50
        else -> 0
    }
}

fn main(): void {
    exit(getValue(25))
}
