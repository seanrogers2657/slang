// Fibonacci sequence
main = () {
    var a: s64 = 0
    var b: s64 = 1
    var c: s64 = 0
    var i = 0
    for ; i < 50; i = i + 1 {
        print(a)
        c = a + b
        a = b
        b = c
    }
}
