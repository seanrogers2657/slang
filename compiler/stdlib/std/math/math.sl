// std/math — basic integer math over s64.
//
// All functions operate on s64 (the default integer). More width-generic
// helpers await implicit literal coercion in binary expressions.

// abs returns the absolute value of x.
// Note: abs(s64 min) overflows and traps, like negating s64 min directly.
abs = (x: s64) -> s64 {
    return if x < 0 { -x } else { x }
}

// min returns the smaller of a and b.
min = (a: s64, b: s64) -> s64 {
    return if a < b { a } else { b }
}

// max returns the larger of a and b.
max = (a: s64, b: s64) -> s64 {
    return if a > b { a } else { b }
}

// sign returns -1, 0, or 1 for negative, zero, and positive x respectively.
sign = (x: s64) -> s64 {
    return if x > 0 { 1 } else if x < 0 { -1 } else { 0 }
}

// clamp returns x constrained to the inclusive range [lo, hi].
clamp = (x: s64, lo: s64, hi: s64) -> s64 {
    return if x < lo { lo } else if x > hi { hi } else { x }
}
