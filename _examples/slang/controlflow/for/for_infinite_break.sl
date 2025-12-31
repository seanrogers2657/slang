// @test: exit_code=42
main = () {
    var count = 0
    for ;; {
        count = count + 1
        if count == 42 {
            exit(count)
        }
    }
}
