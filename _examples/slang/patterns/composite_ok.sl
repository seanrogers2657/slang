// @test: exit_code=0
// @test: stdout=10\n
// Composite pattern. A node cannot own child nodes via *Node fields, so the
// tree is an arena: parallel vecs hold each node's value and child indices
// (-1 = no child). One scope owns the arena and frees it in bulk.
sum_tree = (value: vec, left: vec, right: vec, i: s64) -> s64 {
    if i == 0 - 1 {
        return 0
    }
    return get(value, i)
        + sum_tree(value, left, right, get(left, i))
        + sum_tree(value, left, right, get(right, i))
}

main = () {
    var value = vec()
    var left = vec()
    var right = vec()

    // node 0 = leaf 2, node 1 = leaf 3, node 2 = root 5 with children 0 and 1
    push(value, 2)
    push(left, 0 - 1)
    push(right, 0 - 1)
    push(value, 3)
    push(left, 0 - 1)
    push(right, 0 - 1)
    push(value, 5)
    push(left, 0)
    push(right, 1)

    print(sum_tree(value, left, right, 2))   // 2 + 3 + 5 = 10
}
