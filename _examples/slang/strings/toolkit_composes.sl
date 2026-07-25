// @test: exit_code=0
// @test: stdout=5\n-1\ntrue\nfalse\nhello\nHELLO\nname|age|city\n3\n42\n-1\n
// substr + chr + len + s[i] + concat are together enough to write real string
// manipulation in Slang itself, with no further compiler support.

// index_of returns the byte offset of `sub` in `s` at or after `from`,
// or -1 if it does not occur.
index_of = (s: string, sub: string, from: s64) -> s64 {
    val n = len(s)
    val m = len(sub)
    if m == 0 {
        return from
    }
    var i = from
    while i + m <= n {
        if substr(s, i, i + m) == sub {
            return i
        }
        i = i + 1
    }
    return -1
}

starts_with = (s: string, prefix: string) -> bool {
    if len(prefix) > len(s) {
        return false
    }
    return substr(s, 0, len(prefix)) == prefix
}

// trim strips leading and trailing spaces.
trim = (s: string) -> string {
    var start = 0
    var end = len(s)
    while start < end {
        if s[start] != 32 {
            break
        }
        start = start + 1
    }
    while end > start {
        if s[end - 1] != 32 {
            break
        }
        end = end - 1
    }
    return substr(s, start, end)
}

to_upper = (s: string) -> string {
    var out = ""
    for (var i = 0; i < len(s); i = i + 1) {
        val c = s[i]
        // 'a' = 97, 'z' = 122; clearing bit 5 uppercases
        if c >= 97 {
            if c <= 122 {
                out = out + chr(c - 32)
                continue
            }
        }
        out = out + chr(c)
    }
    return out
}

// parse_int reads a non-negative decimal number.
//
// `d - 48` is u8 arithmetic (the literal takes the byte's type), so a byte
// below '0' would underflow and trap. Non-digits are rejected up front rather
// than allowed to reach the subtraction.
parse_int = (s: string) -> s64 {
    var acc = 0
    for (var i = 0; i < len(s); i = i + 1) {
        val d = s[i]
        if d < 48 {
            return -1
        }
        if d > 57 {
            return -1
        }
        acc = acc * 10 + (d - 48)
    }
    return acc
}

main = () {
    val csv = "name,age,city"

    print(index_of(csv, "age", 0))
    print(index_of(csv, "zip", 0))
    print(starts_with(csv, "name"))
    print(starts_with(csv, "age"))

    print(trim("   hello   "))
    print(to_upper("hello"))

    // Split on ',' by walking separator offsets and slicing between them.
    var out = ""
    var count = 0
    var start = 0
    while true {
        val idx = index_of(csv, ",", start)
        val field = if idx < 0 { substr(csv, start, len(csv)) } else { substr(csv, start, idx) }
        if count > 0 {
            out = out + "|"
        }
        out = out + field
        count = count + 1
        if idx < 0 {
            break
        }
        start = idx + 1
    }
    print(out)
    print(count)

    print(parse_int("42"))
    print(parse_int("4x2"))
}
