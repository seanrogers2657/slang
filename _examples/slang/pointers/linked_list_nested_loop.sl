// @test: exit_code=0
// @test: stdout=batch\nbatch\ndone\n
// Regression test: Building linked list in nested loops
// Tests phi node type inference with *T? in nested loop context

Node = struct {
    var next: *Node?
    var value: s64
}

main = () {
    var head: *Node? = null
    var batch = 0
    while batch < 2 {
        for (var i = 0; i < 3; i = i + 1) {
            val n = new Node{ head, i }
            head = n
        }
        print("batch")
        batch = batch + 1
    }
    print("done")
}
