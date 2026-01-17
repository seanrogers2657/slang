// Quick memory allocation test for profiling
// Allocates 50,000 nodes with 100µs sleep to visualize arena growth
// Run with: go run cmd/slprof/main.go ./build/output

Node = struct {
    var next: *Node?
    var a: i64
    var b: i64
    var c: i64
    var d: i64
}

main = () {
    print("Allocating 50000 nodes with sleep for profiling...")

    var head: *Node? = null
    var x = 0
    while x < 10 {
        var count = 0
        for ; count < 1000000; count = count + 1 {
            val newNode = Heap.new(Node{ head, count, count, count, count })
            head = newNode
            sleep(10 * 1000)
        }
        print("one loop done")
        sleep(20 * 1000)
        x = x + 1
    }

    print("Allocated:")
    print(count)
}
