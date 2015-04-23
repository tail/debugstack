// +build amd64

#define NOSPLIT 4

TEXT ·GetStackPointer(SB),NOSPLIT,$0-8
    MOVQ    SP,8(SP)
    RET
