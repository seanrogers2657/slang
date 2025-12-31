// Fibonacci sequence
main = () {
    var a: i64 = 0
    var b: i64 = 1
    var c: i64 = 0
    var i = 0
    for ; i < 50; i = i + 1 {
        print(a)
        c = a + b
        a = b
        b = c
    }
}
