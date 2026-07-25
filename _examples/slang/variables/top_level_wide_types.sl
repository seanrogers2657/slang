// @test: exit_code=0
// @test: stdout=170141183460469231731687303715884105727\n340282366920938463463374607431768211455\n170141183460469231731687303715884105727\n-170141183460469231731687303715884105728\n
// A global's storage is sized from its declared type. Recording every top-level
// binding as s64 would reserve one word and load one word, silently truncating
// a 128-bit value to its low half.

val S_MAX: s128 = 170141183460469231731687303715884105727
val U_MAX: u128 = 340282366920938463463374607431768211455
var ACC: s128 = 1
var S_MIN: s128 = -170141183460469231731687303715884105728

read_s_max = () -> s128 {
    return S_MAX
}

read_u_max = () -> u128 {
    return U_MAX
}

read_s_min = () -> s128 {
    return S_MIN
}

// Mutating a wide global from another function must write both words
bump = () {
    ACC = ACC + 170141183460469231731687303715884105726
}

main = () {
    print(read_s_max())
    print(read_u_max())

    bump()
    print(ACC)

    print(read_s_min())
}
