// @test: exit_code=1
// @test: stderr_contains=panic: division by zero
fn main(): void {
    val a: i64 = 42
    val b: i64 = 0
    call4(a, b)
}

fn call4(x: int, y: int): int {
    return call3(x, y)
}

fn call3(x: int, y: int): int {
    // comment
    // another one
    // comment
    // another one
    return call2(x, y)
}

fn call2(x: int, y: int): int {
    // comment
    // another one
    return call1(x, y)
}

fn call1(x: int, y: int): int {
    return div(x, y)
}

fn div(x: int, y: int): int {
    return x / y
}
