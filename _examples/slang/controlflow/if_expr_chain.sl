// @test: exit_code=0
// @test: stdout=30\n25\n
// If expressions chained together, nested, used as initializers

abs_val = (x: s64) -> s64 {
    return if x < 0 { 0 - x } else { x }
}

max3 = (a: s64, b: s64, c: s64) -> s64 {
    return if a >= b && a >= c { a } else if b >= c { b } else { c }
}

clamp = (x: s64, lo: s64, hi: s64) -> s64 {
    return if x < lo { lo } else if x > hi { hi } else { x }
}

main = () {
    // Nested if expression
    val a = if true { if false { 1 } else { 2 } } else { 3 }
    assert(a == 2, "nested if expr should be 2")

    // Sum of absolute values 0..5 plus abs(-1..-5)
    var sum: s64 = 0
    for (var i = 0; i <= 5; i = i + 1) {
        sum = sum + abs_val(i)
    }
    for (var i = 1; i <= 5; i = i + 1) {
        sum = sum + abs_val(0 - i)
    }
    // 0+1+2+3+4+5 + 1+2+3+4+5 = 30
    assert(sum == 30, "total sum should be 30")
    print(sum)

    assert(max3(1, 2, 3) == 3, "max3(1,2,3) should be 3")
    assert(max3(3, 1, 2) == 3, "max3(3,1,2) should be 3")
    assert(max3(2, 3, 1) == 3, "max3(2,3,1) should be 3")
    assert(max3(5, 5, 5) == 5, "max3(5,5,5) should be 5")

    assert(clamp(0, 10, 100) == 10, "clamp below")
    assert(clamp(50, 10, 100) == 50, "clamp middle")
    assert(clamp(200, 10, 100) == 100, "clamp above")

    // If expression in arithmetic
    val bonus = 10 + if true { 15 } else { 0 }
    assert(bonus == 25, "bonus should be 25")
    print(bonus)

    // Nested if expr
    val x = 7
    val tier = if x > 5 { if x > 10 { 3 } else { 2 } } else { if x > 2 { 1 } else { 0 } }
    assert(tier == 2, "tier should be 2")
}
