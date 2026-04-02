Counter = class {
    var count: s64

    increment = (self: &&Counter) {
        self.count = self.count + 1
    }

    get = (self: &Counter) -> s64 {
        return self.count
    }
}

classify = (n: s64) -> string {
    return when {
        n > 10  -> "high"
        n > 5   -> "mid"
        else    -> "low"
    }
}

main = () {
    val c = new Counter{ 0 }
    for (var i = 0; i < 12; i = i + 1) {
        c.increment()
    }
    print(classify(c.get()))  // "high"

    val nums = [3, 7, 15, 2]
    var found: s64? = null
    for (var i = 0; i < len(nums); i = i + 1) {
        if nums[i] > 10 && found == null {
            found = nums[i]
        }
    }
    val result = found ?: 0
    print(result)  // 15
}
