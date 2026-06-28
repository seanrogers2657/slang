// @test: exit_code=0
// @test: stdout=3\n100\n
// Passing a vec to a function lets the function mutate it in place: a vec value
// is a handle to a stable header, so the caller sees the appended elements.
fill = (v: vec, n: s64) {
    var i = 0
    while i < n {
        push(v, i * 100)
        i = i + 1
    }
}

main = () {
    var v = vec()
    fill(v, 3)
    print(len(v))      // 3
    print(get(v, 1))   // 100
}
