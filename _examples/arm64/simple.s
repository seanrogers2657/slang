; as -arch arm64 _examples/arm64/simple.s
; ld -o test simple.o -lSystem -syslibroot `xcrun -sdk macosx --show-sdk-path` -e _start -arch arm64
.global _start

_start:
    mov x0, #1
    mov x16, #1
    svc #0
