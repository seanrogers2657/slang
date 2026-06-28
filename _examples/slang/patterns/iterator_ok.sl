// @test: exit_code=0
// @test: stdout=10\n20\n30\n
// Iterator pattern. A cursor cannot hold &vec (a borrow can't be a field), so
// iteration is external: the caller keeps the index and indexes the collection.
main = () {
    var v = vec()
    push(v, 10)
    push(v, 20)
    push(v, 30)

    var i = 0
    while i < len(v) {
        print(get(v, i))
        i = i + 1
    }
}
