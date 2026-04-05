// @test: exit_code=0
// @test: stdout=42\n
// Deeply nested control flow: for inside while inside if inside when

main = () {
    var result: s64 = 0
    val target = 42

    for (var i = 0; i < 5; i = i + 1) {
        var j = 0
        while j < 4 {
            val contribution = when {
                i == 0 && j == 0 -> 1
                i + j > 5 -> 0
                else -> {
                    if i % 2 == 0 {
                        if j % 2 == 0 {
                            1
                        } else {
                            2
                        }
                    } else {
                        if j < 2 {
                            3
                        } else {
                            0
                        }
                    }
                }
            }

            result = result + contribution
            j = j + 1

            if result >= target {
                break
            }
        }

        if result >= target {
            break
        }
    }

    // If we didn't hit target, fill up the rest
    while result < target {
        result = result + 1
    }

    print(result)
}
