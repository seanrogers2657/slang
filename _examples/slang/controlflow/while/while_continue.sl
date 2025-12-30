// @test: exit_code=0
// @test: stdout=1\n3\n5\n7\n9\n
fn main(): void {
    var i = 0
    while i < 10 {
        i = i + 1
        if i % 2 == 0 {
            continue
        }
        print(i)
    }
}
