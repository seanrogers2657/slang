// Bubble Sort - tests array mutation, nested loops, swapping, and verification
// TRIGGERS: Bug 4 (arrays as function params), Bug 5 (unary minus)

is_sorted = (arr: &s64[], size: s64) -> bool {
    for (var i = 0; i < size - 1; i = i + 1) {
        if arr[i] > arr[i + 1] {
            return false
        }
    }
    return true
}

print_array = (arr: &s64[], size: s64) {
    for (var i = 0; i < size; i = i + 1) {
        print(arr[i])
    }
}

main = () {
    // Reverse sorted with duplicates and negatives
    var arr = [99, 42, 42, 17, 9, 5, 3, 1, 0, -3, -7, -42]
    val size = len(arr)

    assert(!is_sorted(arr, size), "array should start unsorted")

    // Bubble sort with early-exit optimization
    var swapped = true
    var pass = 0
    while swapped {
        swapped = false
        for (var i = 0; i < size - 1 - pass; i = i + 1) {
            if arr[i] > arr[i + 1] {
                val temp = arr[i]
                arr[i] = arr[i + 1]
                arr[i + 1] = temp
                swapped = true
            }
        }
        pass = pass + 1
    }

    assert(is_sorted(arr, size), "array should be sorted after bubble sort")

    // Verify sum preserved: 99+42+42+17+9+5+3+1+0+(-3)+(-7)+(-42) = 166
    var sum: s64 = 0
    for (var i = 0; i < size; i = i + 1) {
        sum = sum + arr[i]
    }
    assert(sum == 166, "sum should be preserved after sort")

    assert(arr[0] == -42, "first element should be -42")
    assert(arr[size - 1] == 99, "last element should be 99")

    print("Sorted array:")
    print_array(arr, size)
    print("Bubble sort test passed!")
}
