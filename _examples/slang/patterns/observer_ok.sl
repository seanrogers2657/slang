// @test: exit_code=0
// @test: stdout=7\n7\n
// Observer pattern. You cannot store references to observers (a borrow can't be
// a field, and vec holds s64). Instead observers live as rows in a vec the
// subject owns; "notify" writes each row. Identity is the row index.
notify_all = (observers: vec, value: s64) {
    var i = 0
    while i < len(observers) {
        set(observers, i, value)
        i = i + 1
    }
}

main = () {
    var observers = vec()   // each element is one observer's last-seen value
    push(observers, 0)      // observer A
    push(observers, 0)      // observer B
    notify_all(observers, 7)
    print(get(observers, 0))
    print(get(observers, 1))
}
