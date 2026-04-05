// References format_num from format.sl — same package, no import needed
double_format = (n: s64) -> s64 {
    return format_num(n) + format_num(n)
}
