// @test: exit_code=0
// @test: stdout=1\n2\n3\n10\n20\n30\n
// Pointer collection example demonstrating *T for managing multiple heap objects
// (Self-referential linked list requires forward declaration support)

Point = struct {
    val x: s64
    val y: s64
}

// Create a point on the heap
createPoint = (x: s64, y: s64) -> *Point {
    return Heap.new(Point{ x, y })
}

// Sum and print a point
printPoint = (p: &Point) {
    print(p.x)
}

main = () {
    // Create three separate points
    val p1 = createPoint(1, 10)
    val p2 = createPoint(2, 20)
    val p3 = createPoint(3, 30)

    // Print x values
    printPoint(p1)
    printPoint(p2)
    printPoint(p3)

    // Print y values
    print(p1.y)
    print(p2.y)
    print(p3.y)
}
