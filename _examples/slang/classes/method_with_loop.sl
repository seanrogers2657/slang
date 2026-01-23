// @test: exit_code=55
// Test method containing a loop

Counter = class {
    var value: i64

    // Method with while loop - sum 1 to n
    sumTo = (self: &Counter, n: i64) -> i64 {
        var sum: i64 = 0
        var i: i64 = 1
        while i <= n {
            sum = sum + i
            i = i + 1
        }
        return sum
    }

    // Method with for loop - factorial
    factorial = (self: &Counter, n: i64) -> i64 {
        var result: i64 = 1
        for (var i = 2; i <= n; i = i + 1) {
            result = result * i
        }
        return result
    }
}

main = () {
    val c = Heap.new(Counter{ 0 })
    val sum = c.sumTo(10)  // 1+2+...+10 = 55
    exit(sum)
}
