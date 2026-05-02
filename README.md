# Slang
A compiled programming language targeting ARM64 macOS, with more platforms coming soon.
Slang compiles to native ARM64 assembly and produces standalone executables with no
runtime dependencies. It features a lightweight ownership model, a rich type system,
and helpful compiler errors.

S-lang stands for simple language, super language, stupid language, or shit language,
depending on your mood and whether it's doing what you want. My goal was to build a
language that sits between lower-level languages like C, C++, and Rust and higher-level
languages like Kotlin and Java — ergonomic to use and capable of expressing high-level
concepts, while still being performant when you need to worry about the bits.

> **Note:** Slang is in early alpha. The language, APIs, and tooling are under
> active development and may change without notice. Not recommended for production use.

## Hello World
[`_programs/hello/main.sl`](_programs/hello/main.sl)

```slang
main = () {
    print("Hello, world!")
}
```

## Advanced
[`_programs/advanced/main.sl`](_programs/advanced/main.sl)

```slang
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
```

Explore the `_examples/slang/` folder to learn more about how the language works.
You'll find additional advanced examples in `_programs/`.

## Getting Started

### Requirements

- Go 1.24+
- macOS on Apple Silicon (ARM64)

### Build the `sl` Binary

```bash
go build -o sl cmd/sl/main.go
```

### Compile and Run

```bash
# Compile and run in one step
./sl run _programs/hello/main.sl

# Or compile to a standalone binary
./sl build _programs/hello/main.sl
./build/output
```
