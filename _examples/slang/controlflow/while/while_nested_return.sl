// @test: exit_code=0
// @test: stdout=0\n1\n2\n3\n4\n10\n11\nfound\n
// Test that return exits the entire function from nested loops
// Prints i*10+j until i==1 && j==1, then returns early
fn main(): void {
    val result = search()
    if result {
        print("found")
    } else {
        print("not found")
    }
}

fn search(): bool {
    var i = 0
    while i < 5 {
        var j = 0
        while j < 5 {
            print(i * 10 + j)
            if i == 1 && j == 1 {
                return true  // early return exits both loops and function
            }
            j = j + 1
        }
        i = i + 1
    }
    return false
}
