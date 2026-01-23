// Tests ownership transfer with *s64
// Demonstrates that ownership can be transferred multiple times

transfer = (s: *s64) -> *s64 {
    return s
}

main = () {
    var a = Heap.new(2)

    a = transfer(a)
    sleep(1000 * 1000 * 1000)
    a = transfer(a)
    sleep(1000 * 1000 * 1000)
    a = transfer(a)
    sleep(1000 * 1000 * 1000)
    a = transfer(a)
    sleep(1000 * 1000 * 1000)
    a = transfer(a)
    sleep(1000 * 1000 * 1000)
    a = transfer(a)

    // Pointer still valid after multiple transfers
    // Memory automatically freed when a goes out of scope
}
