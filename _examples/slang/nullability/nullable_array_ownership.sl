// @test: exit_code=0
// @test: stdout=q1\nb\nq1\n2\n6\n6\nfalse false\n
// A nullable array (T[]?) box owns its array: binding from another binding
// deep-copies it, and reassignment, discard, and scope exit free the array and
// its elements. A function's array return size is fixed by its returns.
mk = (b: bool) -> string[]? {
    if b { return ["a${1}", "b"] }
    return null
}
nums = (b: bool) -> s64[]? {
    if b { return [1, 2, 3] }
    return null
}
main = () {
    val a: string[]? = ["q${1}", "r"]
    val b = mk(true)
    val c: string[]? = a
    var d = mk(false)
    d = mk(true)
    d = a
    d = null
    d = ["s${2}", "t"]
    val e = a ?: ["z", "w"]
    print(e[0])
    val f = mk(true) ?: ["y", "x"]
    print(f[1])
    val g = mk(false) ?: e
    print(g[0])
    var arr = [mk(true), mk(false)]
    arr[1] = a
    arr[0] = null
    val h = arr
    print(len(mk(true) ?: ["k", "l"]))
    val t = nums(true) ?: [0, 0, 0]
    print(t[0] + t[1] + t[2])
    val u = nums(false) ?: [4, 5, 6]
    print(u[2])
    mk(true)
    print("${b == null} ${d == null}")
}
