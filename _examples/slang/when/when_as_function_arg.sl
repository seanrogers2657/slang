// @test: exit_code=0
// @test: stdout=1\n0\n2\n
// When expressions used directly as function arguments

report = (code: s64) {
    print(code)
}

main = () {
    val x = 100
    val y = 5

    // When expression directly as function argument
    report(when {
        x > 50 -> 1
        else -> 0
    })

    report(when {
        y > 50 -> 1
        else -> 0
    })

    // Nested when with numeric conditions
    val size = when {
        x > 50 -> 2
        else -> 1
    }
    val importance = when {
        y < 10 -> 1
        else -> 2
    }
    report(when {
        size == 2 && importance == 1 -> 2
        else -> 0
    })
}
