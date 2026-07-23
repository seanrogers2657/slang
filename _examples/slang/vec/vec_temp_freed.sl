// @test: exit_code=0
// @test: stdout=1\n1\n1\n
// Regression: fresh vec temporaries that no binding owns must be freed after
// use — a discarded vec-returning call, a vec temp passed as an argument, and
// a vec temp consumed by len(). Each used to leak its header+buffer and abort
// the runtime's balanced-heap check.
make_vec = () -> vec {
    var v = vec()
    push(v, 5)
    return v
}

use = (v: vec) -> s64 {
    return len(v)
}

main = () {
    make_vec()               // discarded result: freed, not leaked
    print(1)
    print(use(make_vec()))   // 1 — argument temp freed after the call
    print(len(make_vec()))   // 1 — len() operand temp freed after the read
}
