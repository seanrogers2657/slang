// @test: exit_code=0
// @test: stdout=42\n43\n42\n
// Various loop patterns: sum with early exit, skip with continue,
// nested loop with break

main = () {
    // Pattern 1: Sum even numbers, break when sum exceeds 40
    var sum: s64 = 0
    for (var i = 0; i < 100; i = i + 1) {
        if i % 2 != 0 {
            continue
        }
        sum = sum + i
        if sum > 40 {
            break
        }
    }
    // 0+2+4+6+8+10=30, then +12=42 > 40, break
    assert(sum == 42, "sum should be 42")
    print(sum)

    // Pattern 2: Count how many numbers 1-100 are divisible by 3 or 7
    // div3: 33, div7: 14, div21: 4 -> 33+14-4 = 43
    var count: s64 = 0
    for (var i = 1; i <= 100; i = i + 1) {
        if i % 3 == 0 || i % 7 == 0 {
            count = count + 1
        }
    }
    assert(count == 43, "count should be 43")
    print(count)

    // Pattern 3: Nested loop - find first pair (i,j) where i*j == 42
    var found_i: s64 = 0
    var found_j: s64 = 0
    var found = false
    for (var i = 1; i <= 10; i = i + 1) {
        for (var j = 1; j <= 10; j = j + 1) {
            if i * j == 42 {
                found_i = i
                found_j = j
                found = true
                break
            }
        }
        if found {
            break
        }
    }
    // First match: i=6, j=7 -> 42
    assert(found_i == 6, "found_i should be 6")
    assert(found_j == 7, "found_j should be 7")
    print(found_i * found_j)
}
