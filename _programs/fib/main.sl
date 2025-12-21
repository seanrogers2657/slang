fn main() {
    var a: i64 = 0
    var b: i64 = 1
    var c: i64 = 0
    for (var i = 0; i < 50; i = i + 1) {
        print(a)
        c = a + b
        a = b
        b = c
    }
}
