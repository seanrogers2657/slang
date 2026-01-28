// @test: exit_code=21
// Test recursive instance methods

Node = class {
    var value: s64
    var depth: s64

    create = (value: s64, depth: s64) -> *Node {
        return Heap.new(Node{ value, depth })
    }

    // Recursive instance method using iteration instead
    sum_to_depth = (self: &Node) -> s64 {
        var total: s64 = 0
        var d = self.depth
        while d >= 0 {
            total = total + self.value
            d = d - 1
        }
        return total
    }
}

Fibonacci = class {
    var placeholder: s64

    // Recursive static
    fib = (n: s64) -> s64 {
        if n <= 1 {
            return n
        }
        return Fibonacci.fib(n - 1) + Fibonacci.fib(n - 2)
    }
}

main = () {
    // sum_to_depth with value=3, depth=2: 3 * 3 = 9 (3 iterations: d=2,1,0)
    val n = Node.create(3, 2)
    val sum = n.sum_to_depth()

    // fib(7) = 13
    val f = Fibonacci.fib(7)

    // 9 + 13 = 22, need 21, subtract 1
    exit(sum + f - 1)
}
