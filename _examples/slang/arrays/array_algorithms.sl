// @test: exit_code=0
// @test: stdout=7\n0\n7\n22\n1\n
// Array algorithms: min, max, range, sum, linear search, count
// Tests arrays passed as function parameters via &s64[]

find_min = (arr: &s64[], size: s64) -> s64 {
    var min_val = arr[0]
    for (var i = 1; i < size; i = i + 1) {
        if arr[i] < min_val {
            min_val = arr[i]
        }
    }
    return min_val
}

find_max = (arr: &s64[], size: s64) -> s64 {
    var max_val = arr[0]
    for (var i = 1; i < size; i = i + 1) {
        if arr[i] > max_val {
            max_val = arr[i]
        }
    }
    return max_val
}

count_matches = (arr: &s64[], size: s64, target: s64) -> s64 {
    var count: s64 = 0
    for (var i = 0; i < size; i = i + 1) {
        if arr[i] == target {
            count = count + 1
        }
    }
    return count
}

main = () {
    val data = [3, 0, 4, 1, 5, 0, 2, 7, 0, 0]
    val size = len(data)

    val max = find_max(data, size)
    val min = find_min(data, size)
    val range = max - min

    assert(max == 7, "max should be 7")
    assert(min == 0, "min should be 0")
    assert(range == 7, "range should be 7")

    print(max)
    print(min)
    print(range)

    // Sum: 3+0+4+1+5+0+2+7+0+0 = 22
    var sum: s64 = 0
    for (var i = 0; i < size; i = i + 1) {
        sum = sum + data[i]
    }
    assert(sum == 22, "sum should be 22")
    print(sum)

    // Count occurrences of 7
    assert(count_matches(data, size, 7) == 1, "should have one 7")
    print(count_matches(data, size, 7))
}
