// @test: exit_code=0
// @test: stdout=0\n1\n10\n11\n20\n21\n
// Test that break only exits the innermost loop
// Output: i*10 + j for each iteration before inner break
fn main(): void {
    var i = 0
    while i < 3 {
        var j = 0
        while j < 5 {
            if j == 2 {
                break  // breaks inner loop only, outer continues
            }
            print(i * 10 + j)
            j = j + 1
        }
        i = i + 1
    }
}
