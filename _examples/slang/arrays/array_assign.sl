// @test: exit_code=0
// @test: stdout=100\n
fn main(): void {
    var arr = [1, 2, 3]
    arr[0] = 100
    print(arr[0])
}
