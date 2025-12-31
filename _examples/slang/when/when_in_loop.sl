// @test: exit_code=13
// Tests when inside a for loop with assignment statements
main = () {
    var sum = 0
    for var i = 0; i < 5; i = i + 1 {
        when {
            i > 2 -> sum = sum + 5
            else -> sum = sum + 1
        }
    }
    exit(sum)
}
