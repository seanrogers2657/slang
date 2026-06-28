// @test: exit_code=0
// @test: stdout=5\n
// Visitor pattern (AST eval). No interfaces/virtual dispatch, so node kind is a
// tag and `when` does the dispatch. Nodes live in an arena (parallel vecs):
//   kind 0 = literal: a = value
//   kind 1 = add:     a = left index, b = right index
eval = (kind: vec, a: vec, b: vec, i: s64) -> s64 {
    return when {
        get(kind, i) == 0 -> get(a, i)
        else -> eval(kind, a, b, get(a, i)) + eval(kind, a, b, get(b, i))
    }
}

main = () {
    var kind = vec()
    var a = vec()
    var b = vec()

    // node 0 = literal 2
    push(kind, 0)
    push(a, 2)
    push(b, 0)
    // node 1 = literal 3
    push(kind, 0)
    push(a, 3)
    push(b, 0)
    // node 2 = add(0, 1)
    push(kind, 1)
    push(a, 0)
    push(b, 1)

    print(eval(kind, a, b, 2))   // 2 + 3 = 5
}
