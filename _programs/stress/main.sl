// Stress Test: Complex Feature Interactions
// TRIGGERS: Bug 1 (!obj.method), Bug 3 (if obj.field), Bug 6 (bool ==)

Vec2 = struct {
    var x: s64
    var y: s64
}

Particle = class {
    var pos: *Vec2
    var vel: *Vec2
    var alive: bool
    var age: s64

    create = (x: s64, y: s64, vx: s64, vy: s64) -> *Particle {
        return new Particle{
            new Vec2{ x, y },
            new Vec2{ vx, vy },
            true,
            0
        }
    }

    tick = (self: &&Particle) {
        if !self.alive {
            return
        }
        self.pos.x = self.pos.x + self.vel.x
        self.pos.y = self.pos.y + self.vel.y
        self.age = self.age + 1

        if self.pos.x < 0 || self.pos.x > 1000 || self.pos.y < 0 || self.pos.y > 1000 {
            self.alive = false
        }
    }

    classify = (self: &Particle) -> s64 {
        if !self.alive { return 0 }
        return when {
            self.age < 10 -> 1
            self.age < 50 -> 2
            else -> 3
        }
    }

    get_quadrant = (self: &Particle) -> s64 {
        if !self.alive {
            return 0
        }
        return when {
            self.pos.x < 500 && self.pos.y < 500 -> 1
            self.pos.x >= 500 && self.pos.y < 500 -> 2
            self.pos.x < 500 && self.pos.y >= 500 -> 3
            else -> 4
        }
    }

    distance_sq = (self: &Particle) -> s64 {
        return self.pos.x * self.pos.x + self.pos.y * self.pos.y
    }
}

main = () {
    val p1 = Particle.create(100, 100, 5, 3)
    val p2 = Particle.create(500, 500, 10, 10)
    val p3 = Particle.create(990, 990, 5, 5)

    for (var tick = 0; tick < 20; tick = tick + 1) {
        p1.tick()
        p2.tick()
        p3.tick()
    }

    // p1: 100+5*20=200, 100+3*20=160 -> alive
    assert(p1.alive, "p1 should survive 20 ticks")
    assert(p1.pos.x == 200, "p1 x should be 200")
    assert(p1.pos.y == 160, "p1 y should be 160")
    assert(p1.age == 20, "p1 age should be 20")

    // p3: goes out of bounds quickly
    assert(p3.alive == false, "p3 should be dead")

    assert(p1.classify() == 2, "p1 should be mature")
    assert(p3.classify() == 0, "p3 should be dead")

    assert(p1.get_quadrant() == 1, "p1 should be in quadrant 1")
    assert(p3.get_quadrant() == 0, "dead particle returns 0")

    print("p1 position:")
    print(p1.pos.x)
    print(p1.pos.y)

    // Run p1 for 80 more ticks
    for (var i = 0; i < 80; i = i + 1) {
        p1.tick()
    }

    if p1.alive {
        assert(p1.age == 100, "p1 age should be 100")
        assert(p1.classify() == 3, "p1 should be old")
        assert(p1.get_quadrant() == 2, "p1 should be in quadrant 2")
    }

    val d = p1.distance_sq()
    assert(d == 520000, "distance squared should be 520000")

    print("Stress test passed!")
}
