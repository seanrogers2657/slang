// @test: exit_code=0
// @test: stdout=true\nfalse\ntrue\n
fn main(): void {
    val arr = [true, false, true]
    print(arr[0])
    print(arr[1])
    print(arr[2])
}
