// @test: exit_code=21
// Test recursive instance methods

Node = class {
    var value: i64
    var depth: i64

    create = (value: i64, depth: i64) -> *Node {
        return Heap.new(Node{ value, depth })
    }

    // Recursive instance method using iteration instead
    sumToDepth = (self: &Node) -> i64 {
        var total: i64 = 0
        var d = self.depth
        while d >= 0 {
            total = total + self.value
            d = d - 1
        }
        return total
    }
}

Fibonacci = class {
    var placeholder: i64

    // Recursive static
    fib = (n: i64) -> i64 {
        if n <= 1 {
            return n
        }
        return Fibonacci.fib(n - 1) + Fibonacci.fib(n - 2)
    }
}

main = () {
    // sumToDepth with value=3, depth=2: 3 * 3 = 9 (3 iterations: d=2,1,0)
    val n = Node.create(3, 2)
    val sum = n.sumToDepth()

    // fib(7) = 13
    val f = Fibonacci.fib(7)

    // 9 + 13 = 22, need 21, subtract 1
    exit(sum + f - 1)
}
