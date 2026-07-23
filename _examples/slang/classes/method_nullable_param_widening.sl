// @test: exit_code=0
// @test: stdout=hi\nn1\nnone\n15\n10\n1\n0\n
// Regression: passing a non-nullable value to a T? parameter of a method must
// wrap it into the nullable box, exactly as a free-function call does. Without
// the wrap the callee received a bare value where it expects a box and
// dereferenced a non-pointer — a segfault. Covers instance methods (string?
// and s64? params, value/interpolation/null) and a static method.
C = class {
    val base: s64

    show = (self: &C, s: string?) {
        print(s ?: "none")
    }

    tag = (self: &C, n: s64?) -> s64 {
        return self.base + (n ?: 0)
    }

    classify = (s: string?) -> s64 {
        if s == null {
            return 0
        }
        return 1
    }
}

main = () {
    val c = new C{ 10 }
    c.show("hi")            // hi — literal widened to string?
    val v = "n${1}"
    c.show(v)              // n1 — variable widened
    c.show(null)           // none

    print(c.tag(5))        // 15 — s64 widened to s64?
    print(c.tag(null))     // 10

    print(C.classify("x")) // 1 — static method, value widened
    print(C.classify(null))// 0
}
