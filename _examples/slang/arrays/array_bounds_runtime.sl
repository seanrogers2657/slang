// @test: exit_code=1
// @test: stderr_contains=array index out of bounds
fn main(): void {
    val arr = [1, 2, 3]
    var i = 5
    print(arr[i])
}
