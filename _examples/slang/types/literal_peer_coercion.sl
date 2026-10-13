// @test: stdout=255\n9\n255\n7\n250\n170141183460469231731687303715884105727\n
// A bare integer literal among peers that must share a type — array
// elements, if/when branch results — takes the type of a typed peer.
pick = (c: bool) -> u8 {
    return if c { 250u8 } else { 3 }
}

main = () {
    val b: u8 = 7
    val arr = [1, 2u8, 255]
    print(arr[2] + 0u8)

    val arr2 = [b, 0, 9]
    print(arr2[2])

    // x is u8, so 6 + 249 stays in range at u8
    val x = if false { b } else { 6 }
    print(x + 249)

    val y = when {
        false -> 1
        b > 3u8 -> b
        else -> 2
    }
    print(y)

    print(pick(true))

    val big = [1, 170141183460469231731687303715884105727s128]
    print(big[1])
}
