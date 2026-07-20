// @test: exit_code=0
// @test: stdout=5\n1\n
// Regression: a fresh vec temporary passed to the vec builtins themselves
// (push/get/set) bypassed the call-site temp freeing and leaked. Each builtin
// now frees an unowned vec temp after using it.
make_vec = () -> vec {
    var v = vec()
    push(v, 5)
    return v
}

main = () {
    print(get(make_vec(), 0))   // 5 — temp freed after the read
    push(make_vec(), 1)          // temp freed after the (pointless) push
    set(make_vec(), 0, 9)        // temp freed after the write
    print(1)
}
