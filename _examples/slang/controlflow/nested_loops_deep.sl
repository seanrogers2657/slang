// @test: exit_code=27
// Test deeply nested loops (3 levels)

main = () {
    var count = 0
    for (var i = 0; i < 3; i = i + 1) {
        for (var j = 0; j < 3; j = j + 1) {
            for (var k = 0; k < 3; k = k + 1) {
                count = count + 1
            }
        }
    }
    // 3 * 3 * 3 = 27
    exit(count)
}
